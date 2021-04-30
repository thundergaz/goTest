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
func (s {{ $.FileName | camelCase }}Suite) Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}() {
{{range $e := $i.NeedMock }}
	gomonkey.ApplyMethod(reflect.TypeOf(&s.{{ $.SourceNickName }}.{{ $e.StructField }}), "{{ $e.FunctionName }}",
		func(_ *{{ $e.ImportAddress }}, _ context.Context{{ $e.RequestString }}) {{ $e.ResponseString }} {
			{{ . | buildContent }}
		})
{{end}}
	convey.Convey("Test{{ $.FileName | camelCase }}_{{ $i.FunctionName }}", s.T(), func() {
		err := s.{{ $.SourceNickName }}.{{ $i.FunctionName }}(context.Background(), {{ $i.RequestBody | setRequest }})
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

type Suite struct {
	suite.Suite
	mock sqlmock.Sqlmock
}

func (s *Suite) SetupSuite() {
	s.mock = utils.SetupSuite()
}

func (s *Suite) BeforeTest(_, _ string) {
	s.mock.ExpectBegin()
	s.mock.ExpectCommit()
}

func (s *Suite) AfterTest(_, _ string) {
	//require.NoError(s.T(), s.mock.ExpectationsWereMet())
}
`