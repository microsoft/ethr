package tcp

import (
	"weavelab.xyz/ethr/client/tools"
	"weavelab.xyz/ethr/ethr"
)

type Tests struct {
	NetTools *tools.Tools
	Logger   ethr.Logger // This is a hack figure out a better way
}
