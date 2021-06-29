package scenario

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
