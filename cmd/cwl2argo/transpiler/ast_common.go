package transpiler

type String string
type Bool bool
type Int int
type Float float32
type Strings []string

type CWLFormat interface {
	isCWLFormat()
}

func (_ String) isCWLFormat()        {}
func (_ Strings) isCWLFormat()       {}
func (_ CWLExpression) isCWLFormat() {}

type CWLNull struct{}
type CWLBool struct{}
type CWLInt struct{}
type CWLLong struct{}
type CWLFloat struct{}
type CWLDouble struct{}
type CWLString struct{}
type CWLFile struct{}
type CWLDirectory struct{}

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

type LoadListingEnum interface {
	isLoadListingEnum()
}
type NoListing struct{}

func (_ NoListing) isLoadListingEnum() {}

type ShallowListing struct{}

func (_ ShallowListing) isLoadListingEnum() {}

type DeepListing struct{}

func (_ DeepListing) isLoadListingEnum() {}

type CWLClass interface {
}

type CWLExpression struct {
	Expression string
}

type CWLExpressionString interface {
	isCWLExpressionString()
}

func (_ String) isCWLExpressionString()        {}
func (_ CWLExpression) isCWLExpressionString() {}

type CWLExpressionBool interface {
	isCWLExpressionBool()
}

func (_ Bool) isCWLExpressionBool()          {}
func (_ CWLExpression) isCWLExpressionBool() {}

type CWLExpressionInt interface {
	isCWLExpressionInt()
}

func (_ Int) isCWLExpressionInt()           {}
func (_ CWLExpression) isCWLExpressionInt() {}

type CWLExpressionNum interface {
	isCWLExpressionNum()
}

func (_ Int) isCWLExpressionNum()           {}
func (_ Float) isCWLExpressionNum()         {}
func (_ CWLExpression) isCWLExpressionNum() {}

type CWLSecondaryFileSchema struct {
	Pattern  CWLExpressionString
	Required CWLExpressionBool
}

type CWLDefinition struct {
	Class   CWLClass
	Version string
	Inputs  int
	Outputs int
}
