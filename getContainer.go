package goTest

import "strings"

// Container 获取正文（eg.:函数正文、结构体正文）的类
type Container struct {
	// 获取的正文包括头信息（头、参数、返回）
	context string
	// 只有正文 -- 大括号之间的内容
	bodyContext []string
	// 当前各括号的开关值
	bracketValue BracketValue
	// 是否进入函数正文
	inBody bool
	// 函数体结束 存在一种进入函数中，但还没有进入函数正文的状态（在函数头中的时候）需要两种状态才能界定
	contextEnd bool
}
// BracketValue 括号的开关值
type BracketValue struct {
	// 大括号
	Brace int
	// 中括号
	Brackets int
	// 小括号
	Parentheses int
}
// UpdateValue 更新括号值
func (s *BracketValue) UpdateValue(bracket string) {
	switch {
	case strings.Contains("()", bracket):
		s.Parentheses += strings.Index("()", bracket) * -2 + 1
	case strings.Contains("[]", bracket):
		s.Brackets += strings.Index("[]", bracket) * -2 + 1
	case strings.Contains("{}", bracket):
		s.Brace += strings.Index("{}", bracket) * -2 + 1
	}
}

// GetValue 获取括号值
func (s *BracketValue) GetValue() int {
	return s.Brace + s.Brackets + s.Parentheses
}

// ParamsGetContext 构建可选参数
type ParamsGetContext struct {
	// 用来提取内容的括号种类 小0 中1 大2
	kind int
	//
}
type Option func(*ParamsGetContext)

func SetBracket(kind int) Option {
	return func(s *ParamsGetContext) {
		switch kind {
		case 0,1,2:
			s.kind = kind
		default:
			s.kind = 2
		}
	}
}

// 通过分隔符"{}"获取正文 -- 针对多行的长文本
func (s *Container) getContext(str string, options ...func(*ParamsGetContext)) {
	bracketArr := [3][2]string{{"(",")"},{"[","]"},{"{","}"}}
	var bracket [2]string
	var srv ParamsGetContext
	SetBracket(2)(&srv)
	// 只有括号值为零时才能进入正文
	for _, option := range options {
		option(&srv)
	}
	// 获取分界的括号组
	bracket = bracketArr[srv.kind]
	bodyStr := ""
	// 有括号时逐字拼入
	for _, v := range str {
		s.context += string(v)
		char := string(v)
		if strings.Contains("{[()]}", char) {
			// 进入时机 -- 括号值是零且遇到第一个分界括号时 必须在未更新括号值之前
			if s.bracketValue.GetValue() == 0 && char == bracket[0] {
				s.inBody = true
				s.bracketValue.UpdateValue(char)
			} else {
				s.bracketValue.UpdateValue(char)
				if s.inBody {
					// 跳出时机 -- 已经进入到正文内 当前值更新后，刚好括号值清零
					if char == bracket[1] {
						if s.bracketValue.GetValue() == 0 {
							// 进入正文以后，关闭第一个括号后结束体收集
							// TODO:函数体后，是不是直接紧跟又一个函数，目前还未判断
							s.inBody = false
							s.contextEnd = true
							break
						}
					}
					// 已经进入 放在进入之前 是为了不把 分界符也算在bodyStr里面
					if s.bracketValue.GetValue() > 0 {
						bodyStr += string(v)
					}
				}
			}
		}
	}
	if len(bodyStr) > 0 {
		s.bodyContext = append(s.bodyContext, bodyStr)
	}
}