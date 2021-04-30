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
	// 程序文件中的方法列表
	FunctionList []FunctionInfo
	// 文件名称
	FileName string
}
type StructInfo struct {
	StructName   string
	StructFields []StructFieldInfo
}
type StructFieldInfo struct {
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
	// 源程序方法的入参
	ResponseBody string
	// 需要打桩的方法
	NeedMock []Monkey
}
type Monkey struct {
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
			if strings.HasSuffix(path, "impl.go") {
				// 查找其它测试文件
				if strings.Contains(path, "test") {
					fmt.Println("测试文件。")
				} else {
					// 生成标准备测试文件名
					testFileName := strings.Replace(path, ".go", "", 1) + "_auto_test.go"
					// 如果文件存在， 则清除测试文件，只会清除自动生成的测试文件
					if _, err := os.Stat(testFileName); os.IsExist(err) {
						removeErr := os.Remove(testFileName)
						utils.MustCheck(removeErr)
					}
					// 创建base测试文件
					testBase := filepath.Dir(path) + "\\base_test.go"
					if _, err := os.Stat(testBase); os.IsNotExist(err) {
						createBaseTest(testBase)
					}
					// 创建测试文件
					f, createErr := os.Create(testFileName)
					if createErr != nil {
						fmt.Println("创建文件出错。")
					} else {
						// 写入内容
						createTestError := buildTestFile(path, f)
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
func buildTestFile(sourcePath string, file *os.File) error {
	outData := TextFileModule{}
	outData.SourceNickName = "upper"
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
		outData.FileName = strings.Split(filepath.Base(sourcePath), ".")[0]
		for {
			s, _, ok := buffer.ReadLine()
			canWrite := true
			// 这种规则要求注释和正文不能在同一行，应尽量避免这种情况
			if strings.Contains(string(s), "//") {
				canWrite = false
			}
			if strings.Contains(string(s), "/*") {
				canWritable = false
			}
			if ok == io.EOF {
				break
			}
			// 不是注释行时，进行分析
			if canWritable && canWrite {
				// 获取package name
				if strings.Contains(string(s), "package") {
					reg := regexp.MustCompile(`(package )(\b.+\b)$`)
					outData.PackageName = reg.ReplaceAllString(string(s), "$2")
				}
				// 分析文件中的结构体信息
				// 获取类文件中结构体名称
				matchStructName, _ := regexp.MatchString(`^type (\b.+\b)( struct.*)$`, string(s))
				if matchStructName {
					// TODO:默认一个程序文件中只有一个结构体，如果有多个，目前只会拿到最后一个
					StructContent = Container{}
					structStart = true
					regStructName := regexp.MustCompile(`^type (\b.+\b)( struct.*)$`)
					outData.SourceStruct.StructName = regStructName.ReplaceAllString(string(s), "$1")
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
								outData.SourceStruct.StructFields = append(outData.SourceStruct.StructFields, fieldInfo)
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
						function.ResponseBody = requestReg.ReplaceAllString(FunctionContext.context, "$3")
						// 找到上游类的方法 -- (上游类字段outData.SourceStruct.StructFields可能有多个上游)
						for _, field := range outData.SourceStruct.StructFields {
							re := regexp.MustCompile(field.FieldName + `\.([\w\W]+?)\(([^\)]*)\)`)
							// 找到该上游类的方法集合 方法可能会有多个
							mockFunctionArr := re.FindAll([]byte(FunctionContext.context), -1)
							for _, v := range mockFunctionArr {
								// 找到方法名称
								// TODO:找到请求参数与返回类型
								reg := regexp.MustCompile(field.FieldName + `\.([\w\W]+?)\(([^\)]*)\)`)
								function.NeedMock = append(function.NeedMock, Monkey{
									StructField:    field.FieldName,
									ImportAddress:  field.ImportAddress,
									RequestString:  reg.ReplaceAllString(string(v), "$2"),
									ResponseString: "error",
									FunctionName:   reg.ReplaceAllString(string(v), "$1"),
								})
							}
						}
						outData.FunctionList = append(outData.FunctionList, function)
						functionStart = false
					}
				}
			}
			if strings.Contains(string(s), "*/") {
				canWritable = true
			}
		}
		// TODO:如果是普通的文件没有结构体名称，也不会匹配到方法列表只会创建一个空文件
		if outData.SourceStruct.StructName != "" {
			// 创建模板数据
			t, err := template.New("testFile").Funcs(FuncMap()).Parse(tpl.TestTemplate)

			if t != nil {
				err = t.Execute(file, outData)
			}
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
	return nil
}

// 创建基础测试文件
func createBaseTest(pathStr string) {
	outData := TextFileModule{}
	basePath := path.Clean(pathStr)
	packageStr := strings.Split(basePath, "\\")
	outData.PackageName = packageStr[len(packageStr)-2]
	f, createErr := os.Create(pathStr)
	utils.MustCheck(createErr)
	// 创建模板数据
	t, err := template.New("testFile").Funcs(FuncMap()).Parse(tpl.BaseTpl)
	if t != nil {
		err = t.Execute(f, outData)
	}
	if err != nil {
		return
	}
	closeErr := f.Close()
	utils.MustCheck(closeErr)
}

// Container 获取正文的类
type Container struct {
	// 获取的正文
	context string
	// 当前的开关值
	bracketValue int
	// 是否进入函数正文
	contextBodyStart bool
	// 函数正文结束
	contextBodyEnd bool
	// 函数体结束
	contextEnd bool
}

// 通过分隔符获取正文
func (s *Container) getContext(str string) {
	bracketsLen := getBracketsLen(str)
	if bracketsLen == 0 {
		// 没有括号时直接拼入
		s.context += str
	} else {
		// 有括号时逐字拼入
		for _, v := range str {
			s.context += string(v)
			char := string(v)
			// 遇到左括号时进入函数正文
			if char == "{" {
				s.contextBodyStart = true
				s.bracketValue += 1
			}
			if char == "}" {
				s.bracketValue -= 1
			}
			// 进入正文以后，关闭第一个括号后结束函数体收集
			// TODO:函数体后，是不是直接紧跟又一个函数，目前还未判断
			if s.contextBodyStart && s.bracketValue == 0 {
				s.contextBodyEnd = true
				s.contextEnd = true
				break
			}
		}
	}
}

// 串中括号数量
func getBracketsLen(context string) int {
	leftReg := regexp.MustCompile("[^{|}]")
	return len(leftReg.ReplaceAllString(context, ""))
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
	out["buildContent"] = BuildContent
	out["setRequest"] = SetRequest
	out["deriveType"] = DeriveType
	return out
}

// BuildContent 目前生成正文需要根据每种类型来自定义
func BuildContent(content Monkey) string {
	return "return nil"
}
func SetRequest(req string) string {
	// TODO:要根据类型区分以下为ExecuteTpl类型时 入参个数与类型基本固定，可以使用简便方法
	// (ctx context.Context, req *pb.QueryContractDetailRequest,	rsp *pb.QueryContractDetailResponse)
	reg := regexp.MustCompile(`.*\*([^,]*).*\*([^)]*).*`)
	return reg.ReplaceAllString(req, "&$1{}, &$2{}")
}

// DeriveType 通过函数正文推导出字段类型字符串
func DeriveType(field string, functionContext string) string {

	return ""
}
