package server

type RawUI struct {
}

func InitRawUI() (*RawUI, error) {
	return &RawUI{}, nil
}

func (u *RawUI) Paint(seconds uint64) {

}

//func (u *RawUI) printTestResults(results []string) {
//	fmt.Printf("[%13s]  %5s  %7s  %7s  %7s  %8s\n", ui.TruncateStringFromStart(results[0], 13), results[1], results[2], results[3], results[4], results[5])
//}
