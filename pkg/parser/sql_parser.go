package parser

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	braceTokenFamily     = "BRACE"
	opTokenFamily        = "OP"
	logicalOpTokenFamily = "LOGICAL"
	columnTokenFamily    = "COLUMN"

	othersTokenFamily      = "OTHERS"
	valueTokenFamily       = "VALUE"
	quotedValueTokenFamily = "QUOTED"
	openBrace              = "OPEN_BRACE"
	closedBrace            = "CLOSED_BRACE"
	comma                  = "COMMA"
	column                 = "COLUMN"
	value                  = "VALUE"
	quotedValue            = "QUOTED_VALUE"
	eq                     = "EQ"
	notEq                  = "NOT_EQ"
	like                   = "LIKE"
	ilike                  = "ILIKE"
	in                     = "IN"
	listOpenBrace          = "LIST_OPEN_BRACE"
	quotedValueInList      = "QUOTED_VALUE_IN_LIST"
	valueInList            = "VALUE_IN_LIST"
	and                    = "AND"
	or                     = "OR"
	not                    = "NOT"
	// Define the names of the tokens to be parsed

	jsonbFamily           = "JSONB"                    // Each JSONB token will be associated to the JSONB family
	jsonbField            = "JSON_FIELD"               // Each JSONB field
	jsonbArrow            = "JSONB_ARROW"              // The JSONB arrow token (->)
	jsonbToString         = "JSONB_TOSTRING"           // The JSONB to-string token (->>)
	jsonbFieldToStringify = "JSONB_FIELD_TO_STRINGIFY" // The field that will contain the `string` value, ie: ->> FIELD
)
const MaximumComplexity = 10

type checkUnbalancedBraces func() error

type DBQuery struct {
	Query        string
	Values       []interface{}
	ValidColumns []string
	ColumnPrefix string
}

// QueryParser - This object is to be used to parse and validate WHERE clauses (only portion after the `WHERE` is supported)
type QueryParser interface {
	Parse(sql string) (*DBQuery, error)
}

type queryParser struct {
	dbqry DBQuery
}

var _ QueryParser = &queryParser{}

// initStateMachine
// This will be our grammar (each Token will eat the spaces after the Token itself):
// Tokens:
// OPEN_BRACE       = (
// CLOSED_BRACE     = )
// COLUMN           = [A-Za-z][A-Za-z0-9_]*
// VALUE            = [^ ^(^)]+
// QUOTED_VALUE     = '([^']|\\')*'
// EQ               = =
// NOT_EQ           = <>
// LIKE             = [Ll][Ii][Kk][Ee]
// ILIKE             = [Ii][Ll][Ii][Kk][Ee]
// AND              = [Aa][Nn][Dd]
// OR               = [Oo][Rr]
// JSONB_ARROW      = ->
// JSONB_TOSTRING   = ->>
// JSONB_FIELD      = '([^']|\\')*' (example: 'manifest'). The syntax is the same of a quoted string.
// JSONB_STRING_FLD = '([^']|\\')*' (example: 'foo'). The syntax is the same of a quoted value.
//
// VALID TRANSITIONS:
// JSONB_ARROW      -> JSONB_FIELD
// JSONB_FIELD      -> JSONB_ARROW | JSONB_TOSTRING
// JSONB_TOSTRING   -> JSONB_STRING_FLD
// JSONB_STRING_FLD -> EQ | NOT_EQ | LIKE | ILIKE | IN | NOT
func (p *queryParser) initStateMachine() (*State, checkUnbalancedBraces) {

	// counts the number of joins
	complexity := 0

	// contains := arrays.Contains[string]

	// This variable counts the open openBraces
	openBraces := 0
	countOpenBraces := func(tok string) error {
		switch tok {
		case "(":
			openBraces++
		case ")":
			openBraces--
		}
		if openBraces < 0 {
			return errors.Errorf("unexpected ')'")
		}
		return nil
	}

	onNewToken := func(token *ParsedToken) error {
		switch token.Family {
		case braceTokenFamily:
			if err := countOpenBraces(token.Value); err != nil {
				return err
			}
			p.dbqry.Query += token.Value
			return nil
		case valueTokenFamily:
			p.dbqry.Query += " ?"
			p.dbqry.Values = append(p.dbqry.Values, token.Value)
			return nil
		case quotedValueTokenFamily:
			p.dbqry.Query += " ?"
			// unescape
			tmp := strings.ReplaceAll(token.Value, `\'`, "'")
			// remove quotes:
			if len(tmp) > 1 {
				tmp = string([]rune(tmp)[1 : len(tmp)-1])
			}
			p.dbqry.Values = append(p.dbqry.Values, tmp)
			return nil
		case logicalOpTokenFamily:
			complexity++
			if complexity > MaximumComplexity {
				return errors.Errorf("maximum number of permitted joins (%d) exceeded", MaximumComplexity)
			}
			p.dbqry.Query += " " + token.Value + " "
			return nil
		case columnTokenFamily:
			// we want column names to be lowercase
			columnName := strings.ToLower(token.Value)
			// if !contains(p.dbqry.ValidColumns, columnName) {
			// 	return fmt.Errorf("invalid column name: '%s', valid values are: %v", token.Value, p.dbqry.ValidColumns)
			// }
			if p.dbqry.ColumnPrefix != "" && !strings.HasPrefix(columnName, p.dbqry.ColumnPrefix+".") {
				columnName = p.dbqry.ColumnPrefix + "." + columnName
			}
			p.dbqry.Query += columnName
			return nil
		default:
			p.dbqry.Query += " " + token.Value
			return nil
		}
	}

	grammar := Grammar{
		Tokens: []TokenDefinition{
			{Name: openBrace, Family: braceTokenFamily, AcceptPattern: `\(`},
			{Name: closedBrace, Family: braceTokenFamily, AcceptPattern: `\)`},
			{Name: column, Family: columnTokenFamily, AcceptPattern: `[A-Za-z][A-Za-z0-9_]*`},
			{Name: value, Family: valueTokenFamily, AcceptPattern: `[^'][^ ^(^)]*`},
			{Name: quotedValue, Family: quotedValueTokenFamily, AcceptPattern: `'([^']|\\')*'`},
			{Name: eq, Family: opTokenFamily, AcceptPattern: `=`},
			{Name: comma, AcceptPattern: `,`},
			{Name: notEq, Family: opTokenFamily, AcceptPattern: `<>`},
			{Name: like, Family: opTokenFamily, AcceptPattern: `[Ll][Ii][Kk][Ee]`},
			{Name: ilike, Family: opTokenFamily, AcceptPattern: `[Ii][Ll][Ii][Kk][Ee]`},
			{Name: in, Family: opTokenFamily, AcceptPattern: `[Ii][Nn]`},
			{Name: listOpenBrace, Family: braceTokenFamily, AcceptPattern: `\(`},
			{Name: quotedValueInList, Family: quotedValueTokenFamily, AcceptPattern: `'([^']|\\')*'`},
			{Name: valueInList, Family: valueTokenFamily, AcceptPattern: `[^'][^ ^(^)]*`},
			{Name: and, Family: logicalOpTokenFamily, AcceptPattern: `[Aa][Nn][Dd]`},
			{Name: or, Family: logicalOpTokenFamily, AcceptPattern: `[Oo][Rr]`},
			{Name: not, Family: logicalOpTokenFamily, AcceptPattern: `[Nn][Oo][Tt]`},
			{Name: jsonbArrow, Family: jsonbFamily, AcceptPattern: `->`},
			{Name: jsonbField, Family: jsonbFamily, AcceptPattern: `'([^']|\\')*'`},
			{Name: jsonbToString, Family: jsonbFamily, AcceptPattern: `->>`},
			{Name: jsonbFieldToStringify, Family: jsonbFamily, AcceptPattern: `'([^']|\\')*'`},
		},
		Transitions: []TokenTransitions{
			{TokenName: StartState, ValidTransitions: []string{column, openBrace}},
			{TokenName: openBrace, ValidTransitions: []string{column, openBrace}},
			{TokenName: column, ValidTransitions: []string{eq, notEq, like, ilike, in, not, jsonbArrow}},
			{TokenName: eq, ValidTransitions: []string{quotedValue, value}},
			{TokenName: notEq, ValidTransitions: []string{quotedValue, value}},
			{TokenName: like, ValidTransitions: []string{quotedValue, value}},
			{TokenName: ilike, ValidTransitions: []string{quotedValue, value}},
			{TokenName: quotedValue, ValidTransitions: []string{or, and, closedBrace, EndState}},
			{TokenName: value, ValidTransitions: []string{or, and, closedBrace, EndState}},
			{TokenName: closedBrace, ValidTransitions: []string{or, and, closedBrace, EndState}},
			{TokenName: and, ValidTransitions: []string{column, openBrace}},
			{TokenName: or, ValidTransitions: []string{column, openBrace}},
			{TokenName: not, ValidTransitions: []string{in}},
			{TokenName: in, ValidTransitions: []string{listOpenBrace}},
			{TokenName: listOpenBrace, ValidTransitions: []string{quotedValueInList, valueInList}},
			{TokenName: quotedValueInList, ValidTransitions: []string{comma, closedBrace}},
			{TokenName: valueInList, ValidTransitions: []string{comma, closedBrace}},
			{TokenName: comma, ValidTransitions: []string{quotedValueInList, valueInList}},
			{TokenName: jsonbArrow, ValidTransitions: []string{jsonbField}},
			{TokenName: jsonbField, ValidTransitions: []string{jsonbArrow, jsonbToString}},
			{TokenName: jsonbToString, ValidTransitions: []string{jsonbFieldToStringify}},
			{TokenName: jsonbFieldToStringify, ValidTransitions: []string{eq, notEq, like, ilike, in, not}},
		},
	}

	start := NewStateMachineBuilder().
		WithGrammar(&grammar).
		OnNewToken(onNewToken).
		Build()

	return start, func() error {
		if openBraces > 0 {
			return fmt.Errorf("EOF while searching for closing brace ')'")
		}

		return nil
	}
}

func (p *queryParser) Parse(sql string) (*DBQuery, error) {
	state, checkBalancedBraces := p.initStateMachine()

	scanner := NewSQLScanner()
	scanner.Init(sql)

	for scanner.Next() {
		if next, err := state.Move(scanner.Token().Value); err != nil {
			return nil, errors.Errorf("[%d] error parsing the filter: %v", scanner.Token().Position+1, err)
		} else {
			state = next
		}
	}

	if !state.Eof() {
		return nil, errors.Errorf(`EOF encountered while parsing string`)
	}

	if err := checkBalancedBraces(); err != nil {
		return nil, err
	}
	p.dbqry.Query = strings.Trim(p.dbqry.Query, " ")
	return &p.dbqry, nil
}

func NewQueryParser(columns ...string) QueryParser {
	query := DBQuery{}
	return &queryParser{dbqry: query}
}
