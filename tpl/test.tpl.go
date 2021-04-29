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
type {{ .FileName | camelCase }}Suite struct {
	Suite
	{{ .SourceNickName }} {{ .SourceStruct.StructName }}
}

func Test{{ .FileName | camelCase }}(t *testing.T) {
	s := new({{ .FileName | camelCase }}Suite)
	suite.Run(t, s)
}
{{range $i := .FunctionList }}
// {{ $i.FunctionName }}
func (s *{{ $.FileName | camelCase }}Suite) Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}() {
{{/* mock 数据 start */}}{{range $e := $i.NeedMock }}
	gomonkey.ApplyMethod(reflect.TypeOf(&s.{{ $.SourceNickName }}.{{ $e.StructField }}), "{{ $e.FunctionName }}",
		func(_ {{/* mock方法的类型 */}}*{{ $e.ImportAddress }}, _ context.Context{{ $e.RequestString }}) {{ $e.ResponseString }} {
			{{ . | buildContent }}
		})
{{/* mock 数据 end */}}{{end}}
	convey.Convey("Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}", s.T(), func() {
		err := s.{{ $.SourceNickName }}.{{ $i.FunctionName }}(context.Background() {{ /* 主函数请求参数 */}})
		convey.So(err, convey.ShouldBeNil)
	})
}
{{end}}
`
var BaseTpl = `package {{ .PackageName }}

import (
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

// SetupSuite
func (s *Suite) SetupSuite() {
}
// BeforeTest
func (s *Suite) BeforeTest(_, _ string) {
}
// AfterTest
func (s *Suite) AfterTest(_, _ string) {
	//require.NoError(s.T(), s.mock.ExpectationsWereMet())
}
`