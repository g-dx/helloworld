package main
import "fmt"

// AST

type Node struct {
	token *Token
	left  *Node
	right *Node
	stats []*Node
	op    int
}

func (n *Node) Add(stat *Node) {
	n.stats = append(n.stats, stat)
}

const (
	opFuncDcl = iota
	opFuncCall
	opStrLit
	opRoot
)

var nodeTypes = map[int]string {
	opFuncDcl : "Func Decl",
	opFuncCall : "Func Call",
	opStrLit : "String Lit",
	opRoot : "<none>",
}

func printTree(n *Node) {
	fmt.Println("\nParse Tree\n")
	printTreeImpl(n, "    ", true)
	fmt.Println()
}

func printTreeImpl(n *Node, prefix string, isTail bool) {
	// Handle current node
	row := "├── "
	if isTail {
		row = "└── "
	}

    if n == nil {
        return
    }
    // Has token?
    val := "none"
    if n.token != nil {
        val = n.token.val
    }

	fmt.Printf("%v%v%v (\u001B[95m%v\u001B[0m)\n", prefix, row, val, nodeTypes[n.op])

	// Handle 0..n-1 children
	row = "|    "
	if isTail {
		row = "   "
	}
	for i := 0; i < len(n.stats)-1; i++ {
		printTreeImpl(n.stats[i], prefix + row, false)
	}

	// Handle n child
	if len(n.stats) > 0 {
		printTreeImpl(n.stats[len(n.stats)-1], prefix + row, true)
	}
}