package argoexpr

import "testing"

func TestEvalBool(t *testing.T) {
	type args struct {
		input string
		env   interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "test parse expression error",
			args: args{
				input: "invalid expression",
				env:   map[string]interface{}{},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "test eval expression false",
			args: args{
				input: " FOO == 1 ",
				env:   map[string]interface{}{"FOO": 2},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "test eval expression true",
			args: args{
				input: " FOO == 1 ",
				env:   map[string]interface{}{"FOO": 1},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvalBool(tt.args.input, tt.args.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvalBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvalBool() got = %v, want %v", got, tt.want)
			}
		})
	}
}
