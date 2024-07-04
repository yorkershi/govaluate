// Package govaluate comment_from_here
// @author yorkershi
// @created 2023-01-11
package govaluate

import (
	"fmt"
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
		{
			name: "test-4",
			args: args{
				expression: `port_num  <= cache_AMOUNT_0_Member_of_Agg_Port`,
				parameters: map[string]any{
					"cache_AMOUNT_0_Member_of_Agg_Port": "14",
					"port_num":                          14,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-5",
			args: args{
				expression: `port_num  <= cache_AMOUNT_0_Member_of_Agg_Port`,
				parameters: map[string]any{
					"cache_AMOUNT_0_Member_of_Agg_Port": "19",
					"port_num":                          14,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-6",
			args: args{
				expression: `a == b`,
				parameters: map[string]any{
					"a": "19",
					"b": 19,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-7",
			args: args{
				expression: `!a == true`,
				parameters: map[string]any{
					"a": false,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-8",
			args: args{
				expression: `[a.b] == 4`,
				parameters: map[string]any{
					"a.b": 4,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-9",
			args: args{
				expression: `!a`,
				parameters: map[string]any{
					"a": false,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-10",
			args: args{
				expression: `mytest("_govalute_fix_cffpp_", a, b, c)`,
				parameters: map[string]any{
					"a": []any{1, 2, 3},
					"b": []any{4, 5, 6},
					"c": []any{7, 8, 9},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-11",
			args: args{
				expression: `mytest(a, b, c)`,
				parameters: map[string]any{
					"a": 1,
					"b": 2,
					"c": 3,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-12",
			args: args{
				expression: `mytest(a)`,
				parameters: map[string]any{
					"a": 1,
					"b": 2,
					"c": 3,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test-13",
			args: args{
				expression: `mytest(b, c)`,
				parameters: map[string]any{
					"a": 1,
					"b": 2,
					"c": []int{1, 2},
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this, err := NewEvaluableExpressionWithFunctions(tt.args.expression, map[string]ExpressionFunction{
				"mytest": mytest,
			})
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

func mytest(args ...any) (any, error) {
	for i, arg := range args {
		fmt.Println(i, arg)

	}
	return true, nil
}
