package cmd

// didToDirName converts DID to directory name (replace : with -)

// maskAddr masks address for display
func maskAddr(address string) string {
	if len(address) <= 10 {
		return "****"
	}
	return address[:6] + "..." + address[len(address)-4:]
}
