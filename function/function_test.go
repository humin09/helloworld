package function

import (
	"reflect"
	"testing"
)

func TestNewStudent(t *testing.T) {
	tests := []struct {
		name    string
		wantStu Student
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotStu := NewStudent(); !reflect.DeepEqual(gotStu, tt.wantStu) {
				t.Errorf("NewStudent() = %v, want %v", gotStu, tt.wantStu)
			}
		})
	}
}

func Test_print(t *testing.T) {
	type args struct {
		name string
		ver  int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case1",
			args: args{
				name: "hello",
				ver:  1,
			},
			want: "hello-001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := print(tt.args.name, tt.args.ver); got != tt.want {
				t.Errorf("print() = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_compare(t *testing.T) {
	t.Run("compare", func(t *testing.T) {
		NewStudent()
	})

}
