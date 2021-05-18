package goTest

import (
	"bufio"
	"fmt"
	"github.com/thundergaz/goTest/tpl"
	"github.com/thundergaz/goTest/utils"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type TextFileModule struct {
	// 所在包名称
	PackageName string
	// 从程序中获取导入依赖列表
	ImportList []string
	// 每个程序中的结构体名称
	SourceStruct StructInfo
	// 注入源程序的别名
	SourceNickName string
	// 程序文件中的方法列表 -- 待测试的方法
	FunctionList []FunctionInfo
	// 文件名称 用于拼接方法名称与包名称
	FileName string
}

// StructInfo 用于收集原方法中的结构体信息
type StructInfo struct {
	// 结构体名称
	StructName   string
	// 结构体中的字段集合
	StructFields []StructFieldInfo
}
type StructFieldInfo struct {
	// 字段名
	FieldName string
	// 上游函数的结构体import后指向
	ImportAddress string
}
type FunctionInfo struct {
	// 源程序的方法名称
	FunctionName string
	// 源程序的方法类型 ExecuteTpl TransactionTpl DbEngine
	RequestType string
	// 源程序方法的入参
	RequestBody string
	// 源程序方法的出参
	ResponseBody string
	// 函数内部大括号之间的内容 需要断行以信息分析覆盖策略
	FunctionBody []string
	// 需要打桩的方法
	NeedMock []Monkey
}

// BuildContent 目前生成正文需要根据每种类型来自定义
func (t *FunctionInfo) BuildContent()  {
	for _, line := range t.FunctionBody {
		// 规则1：如果是请求参数的逻辑 需要按照罗辑每种情况对请求参数进行赋值一次以达到全覆盖，每一次赋值都需要重新断言
		// 规则2：对mock结果的逻辑判断的 需要对每种结果进行一个mock，并紧接着断言一次。
		has, _ := regexp.MatchString(`(if|switch) ([^{]*)`, line)
		// 正文中存在逻辑判断
		if has {
			reg := regexp.MustCompile(`.*(if|switch) ([^{]*).*`)
			// 获取判断逻辑
			obj := reg.ReplaceAllString(line, "$2")
			// 分析是对请求参数的判断还是对mock结果的判断
			// 查看判断对像是否为自定义对像
			fieldStrArr := strings.FieldsFunc(obj, checkSpiltRule)
			fmt.Println(fieldStrArr)
			// 逻辑词
			lWord := reg.ReplaceAllString(line, "$1")
			if lWord == "if" {
				// if 逻辑
			} else {
				// switch 逻辑
			}
		}
	}
}

type Monkey struct {
	// 是否需要mock 如果是请求参数的逻辑判断是不需单独再进行mock的
	MustMock bool
	// 是哪个上游函数 在 源中的字段是什么
	StructField string
	// 可以通过结构体查出，这里生成的时候直接查出存放
	ImportAddress string
	// 要Mock函数的名称
	FunctionName string
	// 函数的参数
	RequestString string
	// 函数的返回体
	ResponseString string
}

func ScanFold() {
	err := filepath.Walk("./",
		func(path string, f os.FileInfo, err error) error {
			// 这里控制生成规则
			if strings.HasSuffix(path, "_need_t.go") {
				// 查找其它测试文件
				if strings.Contains(path, "test") {
					fmt.Println("测试文件。")
				} else {
					// 生成标准备测试文件名
					testFileName := strings.Replace(path, "_need_t.go", "", 1) + "_auto_test.go"
					// 如果文件存在， 则清除测试文件，只会清除自动生成的测试文件
					if _, err := os.Stat(testFileName); os.IsExist(err) {
						removeErr := os.Remove(testFileName)
						utils.MustCheck(removeErr)
					}
					// 创建base测试文件
					testBase := filepath.Dir(path) + "\\base_test.go"
					if _, err := os.Stat(testBase); os.IsNotExist(err) {
						(*TextFileModule).createBaseTest(&TextFileModule{}, testBase)
					}
					// 创建测试文件
					f, createErr := os.Create(testFileName)
					if createErr != nil {
						fmt.Println("创建文件出错。")
					} else {
						// 写入内容
						createTestError := (*TextFileModule).buildTestFile(&TextFileModule{}, path, f)
						utils.MustCheck(createTestError)
						// 关闭文件
						closeErr := f.Close()
						utils.MustCheck(closeErr)
					}
					// 打印出匹配文件
					// fmt.Println("golang file:", path)
				}
			}
			if f == nil {
				return err
			}
			return nil
		})
	if err != nil {
		return
	}
}

// 通过模版生成测试文件
func (t *TextFileModule) buildTestFile(sourcePath string, file *os.File) error {
	t.SourceNickName = "upper"
	var importStart bool
	var importList Container
	var function FunctionInfo
	// 正文处理
	var FunctionContext Container
	// 正文处理
	var StructContent Container
	// 结构体开始标记
	var structStart bool
	// 函数开始标记
	var functionStart bool
	// 打开源程序文件
	f, err := os.OpenFile(sourcePath, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		canWritable := true
		// 读取文件内容
		buffer := bufio.NewReader(f)
		// 获取文件名称
		t.FileName = strings.Split(filepath.Base(sourcePath), ".")[0]
		for {
			s, _, ok := buffer.ReadLine()
			canWrite := true
			// 这种规则要求注释和正文不能在同一行，应尽量避免这种情况
			// TODO:如果一行从开始就是//注释就不用分析，但如果是在行尾进行注释 就需要把注释删除把排注释的内容流下去。
			viewStr := string(s)
			if strings.Contains(viewStr, "//") {
				canWrite = false
			}
			if strings.Contains(viewStr, "/*") {
				canWritable = false
			}
			if ok == io.EOF {
				break
			}
			// 不是注释行时，进行分析
			if canWritable && canWrite {
				// 获取package name, 阀控制只进一次
				if t.PackageName == "" && strings.Contains(viewStr, "package") {
					reg := regexp.MustCompile(`(package )(\b.+\b)$`)
					t.PackageName = reg.ReplaceAllString(string(s), "$2")
				}
				// 存储依赖列表
				if strings.Contains(viewStr, "import") {
					importStart = true
					importList = Container{}
					// TODO:只有一行时import不需要括号
					// 开始找左括号，从左括号后的行开始计入依赖行
				}
				if importStart {
					importList.getContext(string(s), SetBracket(0))
					if importList.contextEnd {
						importStart = false
					}
				}
				// 分析文件中的结构体信息
				// 获取类文件中结构体名称
				matchStructName, _ := regexp.MatchString(`^type (\b.+\b)( struct.*)$`, viewStr)
				if matchStructName {
					// TODO:默认一个程序文件中只有一个结构体，如果有多个，目前只会拿到最后一个
					StructContent = Container{}
					structStart = true
					regStructName := regexp.MustCompile(`^type (\b.+\b)( struct.*)$`)
					t.SourceStruct.StructName = regStructName.ReplaceAllString(string(s), "$1")
				}
				if structStart {
					StructContent.getContext(string(s))
					if StructContent.contextEnd {
						// 得到完整的结构体正文 StructContent.context
						arr := strings.Split(StructContent.context, "{")
						// 获取结构体字段整文
						fieldStr := strings.Replace(arr[1], "}", "", -1)
						// 结构体正文字符串
						fieldStrArr := strings.FieldsFunc(fieldStr, checkSpiltRule)
						var fieldInfo StructFieldInfo
						for i, file := range fieldStrArr {
							if i%2 == 0 {
								fieldInfo = StructFieldInfo{}
								fieldInfo.FieldName = file
							} else {
								fieldInfo.ImportAddress = file
								t.SourceStruct.StructFields = append(t.SourceStruct.StructFields, fieldInfo)
							}
						}
						// 存入信息
						structStart = false
					}
				}
				// 分析程序中的结构体方法
				// 获取程序中的方法列表
				matchFunction, _ := regexp.MatchString(`^func (\(.*\)) (\b.+\b)(\(.*\)?).*$`, string(s))
				// 函数体开始
				if matchFunction {
					FunctionContext = Container{}
					function = FunctionInfo{}
					functionStart = true
					regFunctionName := regexp.MustCompile(`^func (\(.*\)) (\b.+\b)(\(.*\)?).*$`)
					function.FunctionName = regFunctionName.ReplaceAllString(string(s), "$2")
				}
				// 找到函数头后，获取完整的函数正文
				if functionStart {
					FunctionContext.getContext(string(s))
					if FunctionContext.contextEnd {
						// 得到完整的函数正文 FunctionContext.context
						// 从正文中找到方法的类型 -- (以config.开头的方法。)
						regFunctionType := regexp.MustCompile(`.*config\.([\w\W]+?)\..*`)
						function.RequestType = regFunctionType.ReplaceAllString(FunctionContext.context, "$1")
						// 找到入参
						requestReg := regexp.MustCompile(`func (\([^)]*\))[^)]*(\([^)]*\))([^{]*).*`)
						function.RequestBody = requestReg.ReplaceAllString(FunctionContext.context, "$2")
						// 返回体
						function.ResponseBody = requestReg.ReplaceAllString(FunctionContext.context, "$3")
						function.FunctionBody = FunctionContext.bodyContext
						// function.BuildContent()

						// 找到上游类的方法 -- (上游类字段outData.SourceStruct.StructFields可能有多个上游)
						for _, field := range t.SourceStruct.StructFields {
							re := regexp.MustCompile(field.FieldName + `\.([\w\W]+?)\(([^\)]*)\)`)
							// 找到该上游类的方法集合 方法可能会有多个
							mockFunctionArr := re.FindAll([]byte(FunctionContext.context), -1)
							for _, v := range mockFunctionArr {
								// 找到方法名称
								// TODO:找到请求参数与返回类型
								reg := regexp.MustCompile(field.FieldName + `\.([\w\W]+?)\(([^\)]*)\)`)
								function.NeedMock = append(function.NeedMock, Monkey{
									MustMock:       true,
									StructField:    field.FieldName,
									ImportAddress:  field.ImportAddress,
									RequestString:  reg.ReplaceAllString(string(v), "$2"),
									ResponseString: "error",
									FunctionName:   reg.ReplaceAllString(string(v), "$1"),
								})
							}
						}
						t.FunctionList = append(t.FunctionList, function)
						functionStart = false
					}
				}
			}
			if strings.Contains(string(s), "*/") {
				canWritable = true
			}
		}
		// TODO:如果是普通的文件没有结构体名称，也不会匹配到方法列表只会创建一个空文件
		if t.SourceStruct.StructName != "" {
			// 创建模板数据
			s, err := template.New("testFile").Funcs(FuncMap()).Parse(tpl.TestTemplate)

			if s != nil {
				err = s.Execute(file, t)
			}
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
	return nil
}

// 创建基础测试文件
func (t *TextFileModule) createBaseTest(pathStr string) {
	basePath := path.Clean(pathStr)
	packageStr := strings.Split(basePath, "\\")
	t.PackageName = packageStr[len(packageStr)-2]
	f, createErr := os.Create(pathStr)
	utils.MustCheck(createErr)
	// 创建模板数据
	s, err := template.New("testFile").Funcs(FuncMap()).Parse(tpl.BaseTpl)
	if s != nil {
		err = s.Execute(f, t)
	}
	if err != nil {
		return
	}
	closeErr := f.Close()
	utils.MustCheck(closeErr)
}
func checkSpiltRule(r rune) bool {
	// 使用空格或制表符，把字串分隔成字段
	if r < 33 {
		return true
	}
	return false
}
func FuncMap() template.FuncMap {
	out := utils.FuncMap()
	out["setRequest"] = SetRequest
	out["mockFRequest"] = MockFRequest
	out["setResponse"] = SetResponse
	return out
}

func SetResponse(res string) string  {
	resStr := strings.Replace(strings.Replace(res, "(", "", -1), ")", "", -1)
	resArr := strings.Split(resStr, ",")
	var result string
	for i := range resArr {
		if len(resArr) == i + 1 {
			result += "err"
		} else {
			result += "_, "
		}
	}
	return result
}

func SetRequest(req string) string {
	// TODO:要根据类型区分以下为ExecuteTpl类型时 入参个数与类型基本固定，可以使用简便方法
	// (ctx context.Context, req *pb.QueryContractDetailRequest,	rsp *pb.QueryContractDetailResponse)
	reg := regexp.MustCompile(`\b[^ ]*\b (\*)?([^,|)| ]*)`)
	matches := reg.FindAllStringSubmatch(req, -1)
	var res string
	for _, v := range matches {
		point := ""
		addStr := ""
		if strings.Contains(v[2], "Context") {
			continue
		}
		if len(v[1]) > 0 {
			point = "&"
		}
		addStr = ", " + point + v[2] +"{}"
		if v[2] == "int32" || v[2] == "int64" {
			addStr = ", 1"
		}
		res += addStr
	}
	return res
}

// MockFRequest 分析并返回Mock方法的参数
func MockFRequest(functionRequest string, functionBody []string, req string) string {
	var reqStr string
	// 分离参数
	reqArr := strings.Split(req, ",")
	for i, s := range reqArr {
		if i > 0 {
			reqStr += ", "
		}
		reqStr += "_ "
		// 分析请求参数的类型
		isInRequest, _ := regexp.MatchString(`\b`+s+`\b`, functionRequest)
		// 参数存在官于主函数的参数中
		if isInRequest {
			// 获取此参数的类型
			reg := regexp.MustCompile(`.*\b` + s + `\b (\b[^,]*\b).*`)
			reqStr += reg.ReplaceAllString(functionRequest, "$1")
		} else {
			reqStr += s
		}
		// TODO:函数体内部定义变量
		//isInBody, _ := regexp.MatchString("(var \b" + s + "\b|" + "s :\= )", functionBody)
		//if isInBody {
		//	reg := regexp.MustCompile(".*\b" + s + "\b (\b.*\b).*")
		//	reqStr += reg.ReplaceAllString(req, "$1")
		//}
	}

	return reqStr
}
