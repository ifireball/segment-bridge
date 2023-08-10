package queryprint

type visitable interface {
	visit(visitor) string
}

type (
	query   []command
	command struct {
		command string
		args    visitable
	}
	commaSepArgs []visitable
	spaceSepArgs []visitable
	exprElements []visitable
	plainArg     string
)

// We may want to find a way to auto-generate the code below from the type
// definitions above

type visitor interface {
	visitQuery(query) string
	visitCommand(command) string
	visitCommaSepArgs(commaSepArgs) string
	visitSpaceSepArgs(spaceSepArgs) string
	visitExprElements(exprElements) string
	visitPlainArg(plainArg) string
}

func (o query) visit(v visitor) string        { return v.visitQuery(o) }
func (o command) visit(v visitor) string      { return v.visitCommand(o) }
func (o commaSepArgs) visit(v visitor) string { return v.visitCommaSepArgs(o) }
func (o spaceSepArgs) visit(v visitor) string { return v.visitSpaceSepArgs(o) }
func (o exprElements) visit(v visitor) string { return v.visitExprElements(o) }
func (o plainArg) visit(v visitor) string     { return v.visitPlainArg(o) }
