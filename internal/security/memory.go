package security

func ZeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
