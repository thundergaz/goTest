package tpl

var TestTemplate = `package {{ .PackageName }}

import (
	"context"
{{range $l := .ImportList }}
	$l
{{end}}
	"github.com/agiledragon/gomonkey"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/suite"
	"reflect"
	"testing"
)

// {{ .FileName | camelCase }}Suite 测试基本结构体
type {{ .FileName | camelCase }}Suite struct {
	Suite
	{{ .SourceNickName }} {{ .SourceStruct.StructName }}
}
// Test{{ .FileName | camelCase }}
func Test{{ .FileName | camelCase }}(t *testing.T) {
	s := new({{ .FileName | camelCase }}Suite)
	suite.Run(t, s)
}
{{range $i := .FunctionList }}
// Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}
func (s {{ $.FileName | camelCase }}Suite) Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}() {
	{{range $e := $i.NeedMock }}
	gomonkey.ApplyMethod(reflect.TypeOf(s.{{ $.SourceNickName }}.{{ $e.StructField }}), "{{ $e.FunctionName }}",
		func(_ {{ $e.ImportAddress }}, {{ $e.RequestString | mockFRequest $i.RequestBody $i.FunctionBody }}) {{ $e.ResponseString }} {
			return nil
	})
	{{ end }}
	convey.Convey("Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}", s.T(), func() {
		{{ $i.ResponseBody | setResponse }} := s.{{ $.SourceNickName }}.{{ $i.FunctionName }}(context.Background(){{ $i.RequestBody | setRequest }})
		convey.So(err, convey.ShouldBeNil)
	})
}
{{end}}
`
var BaseTpl = `package {{ .PackageName }}

import (
	"git.code.oa.com/soho/finance/infrastructure/utils"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/suite"
)

// 测试基本结构体
type Suite struct {
	suite.Suite
	mock sqlmock.Sqlmock
}
// SetupSuite 单元测试初始化操作
func (s *Suite) SetupSuite() {
	s.mock = utils.SetupSuite()
}

// BeforeTest 单元测试前操作
func (s *Suite) BeforeTest(_, _ string) {
	s.mock.ExpectBegin()
	s.mock.ExpectCommit()
}

// AfterTest 单元测试后操作
func (s *Suite) AfterTest(_, _ string) {
	//require.NoError(s.T(), s.mock.ExpectationsWereMet())
}
`