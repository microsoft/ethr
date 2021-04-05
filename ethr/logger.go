package ethr

type Logger interface {
	Errorf(...interface{})
	Infof(...interface{})
	Debug(...interface{})
}
