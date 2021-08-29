package generate

type Timeslot struct {
	DayOfWeek int // 0 - 6
	Period    int // 1 - 5
}

// PreferredTimeSlots は学生が履修希望するTimeslotsを表すsliceを返す
// period > DayOfWeekで優先度をつける
// period: 3限 > 4限 > 5限 > 2限 > 1限
// DoW : 水 > 火 > 木 > 金 > 月
func PreferredTimeSlots() []Timeslot {
	list := make([]Timeslot, 30)

	preferredPeriods := []int{3, 4, 5, 2, 1}
	preferredDays := []int{3, 2, 4, 5, 1}
	for p := range preferredPeriods {
		for d := range preferredDays {
			list = append(list, Timeslot{DayOfWeek: d, Period: p})
		}
	}
	return list
}
