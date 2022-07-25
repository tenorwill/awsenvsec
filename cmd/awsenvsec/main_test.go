package main

import (
	"testing"
)

func Test_isItJSON(t *testing.T) {
	type args struct {
		jsonString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			args: args{
				jsonString: "testString",
			},
			want: false,
		},
		{
			args: args{
				jsonString: "{\"mySecret\": \"myPassword\"}",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isItJSON(tt.args.jsonString); got != tt.want {
				t.Errorf("isItJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
