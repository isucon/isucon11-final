package scenario

import (
	"hash/crc32"
)

// 講義に重複がなく内容が一致
func equalCourses(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// 重複チェック
	existed := map[string]bool{}
	for i := 0; i < len(actual); i++ {
		if !existed[actual[i]] {
			existed[actual[i]] = true
		} else {
			return false
		}
	}

	// 不一致チェック
	for _, x := range expected {
		var existInActual bool
		for _, y := range actual {
			if y == x {
				existInActual = true
				break
			}
		}
		if !existInActual {
			return false
		}
	}
	return true
}

func getHash(data []byte) []byte {
	h := crc32.ChecksumIEEE(data)
	return []byte{
		byte(h & 0xff),
		byte((h >> 8) & 0xff),
		byte((h >> 16) & 0xff),
		byte((h >> 24) & 0xff)}
}
