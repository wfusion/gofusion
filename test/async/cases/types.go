package cases

const (
	nameDefault   = "default"
	nameWithQueue = "with_queues"
)

type cs struct {
	name    string
	subTest func()
}
