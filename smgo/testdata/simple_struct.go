package simplestruct

type Person struct {
	Name string
}

func (p *Person) SayHi() {
	print("Hi, I'm " + p.Name)
}
