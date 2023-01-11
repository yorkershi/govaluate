//Package govaluate comment_from_here
//@author yorkershi
//@created 2023-01-11
package govaluate

import (
	"reflect"
	"testing"
)

func TestEvaluableExpression_Evaluate(t *testing.T) {
	type args struct {
		expression string
		parameters map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "test-1",
			args: args{
				expression: `result == true`,
				parameters: map[string]any{
					"result": true,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-2",
			args: args{
				expression: `result == "True"`,
				parameters: map[string]any{
					"result": "true",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-3",
			args: args{
				expression: `result == "False"`,
				parameters: map[string]any{
					"result": "false",
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this, err := NewEvaluableExpressionWithFunctions(tt.args.expression, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := this.Evaluate(tt.args.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Evaluate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
