package order

import "testing"

func TestCanTransitionAllowsReissueFromClosed(t *testing.T) {
	cases := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{name: "未支付转已支付", from: StatusPending, to: StatusPaid, want: true},
		{name: "未支付转已关闭", from: StatusPending, to: StatusClosed, want: true},
		{name: "已支付转通知失败", from: StatusPaid, to: StatusNotifyFailed, want: true},
		{name: "通知失败转已支付", from: StatusNotifyFailed, to: StatusPaid, want: true},
		{name: "过期关闭后人工补单", from: StatusClosed, to: StatusPaid, want: true},
		{name: "已关闭转通知失败", from: StatusClosed, to: StatusNotifyFailed, want: false},
		{name: "已支付转已关闭", from: StatusPaid, to: StatusClosed, want: false},
		{name: "无效状态来源", from: Status(9), to: StatusPaid, want: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanTransition(c.from, c.to); got != c.want {
				t.Fatalf("CanTransition(%d, %d) = %v, want %v", c.from, c.to, got, c.want)
			}
		})
	}
}

func TestCanReissueOnlyPendingAndClosed(t *testing.T) {
	cases := []struct {
		status Status
		want   bool
	}{
		{status: StatusPending, want: true},
		{status: StatusClosed, want: true},
		{status: StatusPaid, want: false},
		{status: StatusNotifyFailed, want: false},
		{status: Status(9), want: false},
	}

	for _, c := range cases {
		if got := CanReissue(c.status); got != c.want {
			t.Fatalf("CanReissue(%d) = %v, want %v", c.status, got, c.want)
		}
	}
}
