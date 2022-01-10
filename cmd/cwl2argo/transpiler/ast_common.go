package transpiler

type (
	String         string
	Bool           bool
	Int            int
	Float          float32
	Strings        []string
	SecondaryFiles []CWLSecondaryFileSchema
)

type CWLFormatKind int32

const (
	FormatStringKind CWLFormatKind = iota
	FormatStringsKind
	FormatExpressionKind
)

type CWLFormat struct {
	Kind       CWLFormatKind
	String     String
	Strings    Strings
	Expression CWLExpression
}

type (
	CWLNull      struct{}
	CWLBool      struct{}
	CWLInt       struct{}
	CWLLong      struct{}
	CWLFloat     struct{}
	CWLDouble    struct{}
	CWLString    struct{}
	CWLFile      struct{}
	CWLDirectory struct{}
)

func (_ CWLNull) isCWLType()      {}
func (_ CWLBool) isCWLType()      {}
func (_ CWLInt) isCWLType()       {}
func (_ CWLLong) isCWLType()      {}
func (_ CWLFloat) isCWLType()     {}
func (_ CWLDouble) isCWLType()    {}
func (_ CWLString) isCWLType()    {}
func (_ CWLFile) isCWLType()      {}
func (_ CWLDirectory) isCWLType() {}

type CWLStdin struct{}

type CWLType interface {
	isCWLType()
}

type LoadListingKind int32

const (
	ShallowListingKind LoadListingKind = iota
	DeepListingKind
	NoListingKind
)

type LoadListingEnum struct {
	Kind LoadListingKind
}

type CWLExpressionKind int32

const (
	RawKind CWLExpressionKind = iota
	ExpressionKind
	BoolKind
	IntKind
	FloatKind
)

type CWLExpression struct {
	Kind       CWLExpressionKind
	Raw        string
	Expression string
	Bool       bool
	Int        int
	Float      float64
}

type CWLSecondaryFileSchema struct {
	Pattern  CWLExpression `yaml:"pattern"`
	Required CWLExpression `yaml:"required"`
}
