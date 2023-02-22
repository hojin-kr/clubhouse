package trace

import (
	"io"
)

// Tracer 코드 전체에서 이벤트를 추적할 수 있는 객체를 설명하는 인터페아스
// 대문자 T로 시작한 이유는 공개적으로 보이는 타입임을 의미함
type Tracer interface {
	Trace(...interface{})
}

// New
func New(w io.Writer) Tracer {
	return &tracer{out: w}
}

// tracer의 타입에는 out 이라는 io.Writer 필드가 있으며, 추적 출력에 사용
type tracer struct {
	out io.Writer
}

// Trace 메소드가 호출되면 추적 세부 사항을 형식화해서 out 출력기에 기록
func (t *tracer) Trace(a ...interface{}) {
	// fmt.Fprint(t.out, a...)
	// fmt.Fprintln(t.out)
}

type nilTracer struct{}

func (t *nilTracer) Trace(a ...interface{}) {}

func Off() Tracer {
	return &nilTracer{}
}
