package queryprint

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    []token
		wantErr bool
	}{
		{
			name: "Query #1",
			query: `search index=some_long_index_name log_type=awesome match=value` +
				`|eval custom_field=some_expression,` +
				`other_field=other_expression` +
				`|fields fields,shown,in,results`,
			want: []token{
				{typ: tIDENTIFIER, value: "search"},
				{typ: tSPACE},

				{typ: tIDENTIFIER, value: "index"},
				{typ: tEQUAL},
				{typ: tIDENTIFIER, value: "some_long_index_name"},
				{typ: tSPACE},

				{typ: tIDENTIFIER, value: "log_type"},
				{typ: tEQUAL},
				{typ: tIDENTIFIER, value: "awesome"},
				{typ: tSPACE},

				{typ: tIDENTIFIER, value: "match"},
				{typ: tEQUAL},
				{typ: tIDENTIFIER, value: "value"},

				{typ: tPIPE},
				{typ: tIDENTIFIER, value: "eval"},
				{typ: tSPACE},

				{typ: tIDENTIFIER, value: "custom_field"},
				{typ: tEQUAL},
				{typ: tIDENTIFIER, value: "some_expression"},
				{typ: tCOMMA},

				{typ: tIDENTIFIER, value: "other_field"},
				{typ: tEQUAL},
				{typ: tIDENTIFIER, value: "other_expression"},

				{typ: tPIPE},
				{typ: tIDENTIFIER, value: "fields"},
				{typ: tSPACE},

				{typ: tIDENTIFIER, value: "fields"},
				{typ: tCOMMA},
				{typ: tIDENTIFIER, value: "shown"},
				{typ: tCOMMA},
				{typ: tIDENTIFIER, value: "in"},
				{typ: tCOMMA},
				{typ: tIDENTIFIER, value: "results"},
			},
		},
		{
			name:  "All tokens",
			query: `.=>(<-%|+)/*!===>=<=AND NOT"st|st"OR,XOR.'fld'`,
			want: []token{
				{typ: tDOT},
				{typ: tEQUAL},
				{typ: tGREATER},
				{typ: tLEFT_PAREN},
				{typ: tLESS},
				{typ: tMINUS},
				{typ: tPERCENT},
				{typ: tPIPE},
				{typ: tPLUS},
				{typ: tRIGHT_PAREN},
				{typ: tSLASH},
				{typ: tSTAR},
				{typ: tBANG_EQUAL},
				{typ: tEQUAL_EQUAL},
				{typ: tGREATER_EQUAL},
				{typ: tLESS_EQUAL},
				{typ: tAND},
				{typ: tSPACE},
				{typ: tNOT},
				{typ: tSTRING, value: "\"st|st\""},
				{typ: tOR},
				{typ: tCOMMA},
				{typ: tXOR},
				{typ: tDOT},
				{typ: tQUOTED_FIELD, value: "'fld'"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := make(chan token)
			var got []token
			var err error

			go func() {
				err = ScanQuery(strings.NewReader(tt.query), c)
				close(c)
			}()
			for t := range c {
				got = append(got, t)
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
