package util

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func TestCourseQueue_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		queue func() *courseQueue
		want  bool
	}{
		{
			name: "empty with new",
			queue: func() *courseQueue {
				return newCourseQueue(1)
			},
			want: true,
		},
		{
			name: "empty with 2 push, 2 pop",
			queue: func() *courseQueue {
				q := newCourseQueue(1)
				q.Push(&model.Course{})
				q.Pop()
				return q
			},
			want: true,
		},
		{
			name: "non empty with 2 push, 1 pop",
			queue: func() *courseQueue {
				q := newCourseQueue(1)
				q.Push(&model.Course{})
				return q
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.queue()
			if got := q.isEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCourseQueue_Len(t *testing.T) {
	tests := []struct {
		name  string
		queue func() *courseQueue
		want  int
	}{
		{
			name: "len 0",
			queue: func() *courseQueue {
				return newCourseQueue(10)
			},
			want: 0,
		},
		{
			name: "len 1",
			queue: func() *courseQueue {
				q := newCourseQueue(1)
				q.Push(&model.Course{})
				return q
			},
			want: 1,
		},
		{
			name: "len with grow",
			queue: func() *courseQueue {
				q := newCourseQueue(1)
				q.Push(&model.Course{})
				q.Push(&model.Course{})
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

func TestCourseDeque_Push_Pop(t *testing.T) {
	t.Run("push10, pop10", func(t *testing.T) {
		q := newCourseQueue(10)
		loopCount := 100

		expected := make([]*model.Course, 0, loopCount)
		for i := 0; i < loopCount; i++ {
			a := &model.Course{ID: strconv.FormatInt(int64(i), 10)}
			switch rand.Intn(2) {
			case 0:
				q.Push(a)
				expected = append([]*model.Course{a}, expected...)
			case 1:
				q.Pop()
				if len(expected) > 0 {
					expected = expected[1:]
				}
			}
		}

		if q.Len() != len(expected) {
			t.Fatalf("len = %v, want = %v", q.Len(), len(expected))
		}
		for i := 0; i < q.Len(); i++ {
			a := q.Pop()
			if a == nil {
				t.Fatalf("returned nil")
			}
			if a.ID != expected[i].ID {
				t.Errorf("actual = %v, want %v", a.ID, i)
			}
		}
	})
}
