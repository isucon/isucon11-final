package model

import (
	"math/rand"
	"testing"
)

func TestAnnouncementDeque_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		queue func() *AnnouncementDeque
		want  bool
	}{
		{
			name: "empty with new",
			queue: func() *AnnouncementDeque {
				return NewAnnouncementDeque(1)
			},
			want: true,
		},
		{
			name: "empty with 2 push, 2 pop",
			queue: func() *AnnouncementDeque {
				q := NewAnnouncementDeque(1)
				q.PushBack(&Announcement{})
				q.PushFront(&Announcement{})
				q.PopBack()
				q.PopFront()
				return q
			},
			want: true,
		},
		{
			name: "non empty with 2 push, 1 pop",
			queue: func() *AnnouncementDeque {
				q := NewAnnouncementDeque(1)
				q.PushBack(&Announcement{})
				q.PushFront(&Announcement{})
				q.PopBack()
				return q
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.queue()
			if got := q.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnnouncementDeque_Len(t *testing.T) {
	tests := []struct {
		name  string
		queue func() *AnnouncementDeque
		want  int
	}{
		{
			name: "len 0",
			queue: func() *AnnouncementDeque {
				return NewAnnouncementDeque(10)
			},
			want: 0,
		},
		{
			name: "len 2",
			queue: func() *AnnouncementDeque {
				q := NewAnnouncementDeque(2)
				q.PushFront(&Announcement{})
				q.PushBack(&Announcement{})
				return q
			},
			want: 2,
		},
		{
			name: "len 2",
			queue: func() *AnnouncementDeque {
				q := NewAnnouncementDeque(1)
				q.PushFront(&Announcement{})
				q.PushBack(&Announcement{})
				return q
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.queue()
			if got := q.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnnouncementDeque_PushBack_PopFront(t *testing.T) {
	t.Run("push10, pop10", func(t *testing.T) {
		q := NewAnnouncementDeque(1)
		for i := 0; i < 10; i++ {
			q.PushBack(&Announcement{CreatedAt: int64(i)})
		}
		for i := 0; i < 10; i++ {
			a := q.PopFront()
			if a.CreatedAt != int64(i) {
				t.Errorf("actual = %v, want %v", a.CreatedAt, i)
			}
		}
	})
}

func TestAnnouncementDeque_PushFront_PopBack(t *testing.T) {
	t.Run("push10, pop10", func(t *testing.T) {
		q := NewAnnouncementDeque(1)
		for i := 0; i < 10; i++ {
			q.PushFront(&Announcement{CreatedAt: int64(i)})
		}
		for i := 0; i < 10; i++ {
			a := q.PopBack()
			if a == nil {
				t.Errorf("returned nil")
			}
			if a.CreatedAt != int64(i) {
				t.Errorf("actual = %v, want %v", a.CreatedAt, i)
			}
		}
	})
}

func TestAnnouncementDeque_Push_Pop(t *testing.T) {
	t.Run("push10, pop10", func(t *testing.T) {
		q := NewAnnouncementDeque(10)
		loopCount := 100

		expected := make([]*Announcement, 0, loopCount)
		for i := 0; i < loopCount; i++ {
			a := &Announcement{CreatedAt: int64(i)}
			switch rand.Intn(4) {
			case 0:
				q.PushFront(a)
				expected = append([]*Announcement{a}, expected...)
			case 1:
				q.PushFront(a)
				expected = append(expected, a)
			case 2:
				q.PopBack()
				if len(expected) > 0 {
					expected = expected[:len(expected)-1]
				}
			case 3:
				q.PopFront()
				if len(expected) > 0 {
					expected = expected[1:]
				}

			}
		}

		if q.Len() != len(expected) {
			t.Errorf("len = %v, want = %v", q.Len(), len(expected))
		}
		for i := 0; i < q.Len(); i++ {
			a := q.PopFront()
			if a == nil {
				t.Errorf("returned nil")
			}
			if a.CreatedAt != expected[i].CreatedAt {
				t.Errorf("actual = %v, want %v", a.CreatedAt, i)
			}
		}
	})
}
