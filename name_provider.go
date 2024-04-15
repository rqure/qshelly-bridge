package main

type NameProvider struct{}

func (n *NameProvider) Get() string {
	return "qmq2mqtt"
}
