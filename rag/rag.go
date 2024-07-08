package rag

type Doc struct {
	ID string
	Title string
	Content string
	Context string
	Embedding []float32
}