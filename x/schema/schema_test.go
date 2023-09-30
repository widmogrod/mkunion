package schema

import (
	"fmt"
	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"runtime"
	"testing"
)

type GenerateData struct {
	ID                 string   `faker:"uuid_digit"`
	Latitude           float32  `faker:"lat"`
	Longitude          float32  `faker:"long"`
	CreditCardNumber   string   `faker:"cc_number"`
	CreditCardType     string   `faker:"cc_type"`
	Email              string   `faker:"email"`
	DomainName         string   `faker:"domain_name"`
	IPV4               string   `faker:"ipv4"`
	IPV6               string   `faker:"ipv6"`
	Password           string   `faker:"password"`
	Jwt                string   `faker:"jwt"`
	PhoneNumber        string   `faker:"phone_number"`
	MacAddress         string   `faker:"mac_address"`
	URL                string   `faker:"url"`
	UserName           string   `faker:"username"`
	TollFreeNumber     string   `faker:"toll_free_number"`
	E164PhoneNumber    string   `faker:"e_164_phone_number"`
	TitleMale          string   `faker:"title_male"`
	TitleFemale        string   `faker:"title_female"`
	FirstName          string   `faker:"first_name"`
	FirstNameMale      string   `faker:"first_name_male"`
	FirstNameFemale    string   `faker:"first_name_female"`
	LastName           string   `faker:"last_name"`
	Name               string   `faker:"name"`
	UnixTime           int64    `faker:"unix_time"`
	Date               string   `faker:"date"`
	Time               string   `faker:"time"`
	MonthName          string   `faker:"month_name"`
	Year               string   `faker:"year"`
	DayOfWeek          string   `faker:"day_of_week"`
	DayOfMonth         string   `faker:"day_of_month"`
	Timestamp          string   `faker:"timestamp"`
	Century            string   `faker:"century"`
	TimeZone           string   `faker:"timezone"`
	TimePeriod         string   `faker:"time_period"`
	Word               string   `faker:"word"`
	Sentence           string   `faker:"sentence"`
	Paragraph          string   `faker:"paragraph"`
	Currency           string   `faker:"currency"`
	Amount             float64  `faker:"amount"`
	AmountWithCurrency string   `faker:"amount_with_currency"`
	UUIDHypenated      string   `faker:"uuid_hyphenated"`
	UUID               *string  `faker:"uuid_digit"`
	Skip               string   `faker:"-"`
	PaymentMethod      string   `faker:"oneof: cc, paypal, check, money order"` // oneof will randomly pick one of the comma-separated values supplied in the tag
	AccountID          int      `faker:"oneof: 15, 27, 61"`                     // use commas to separate the values for now. Future support for other separator characters may be added
	Price32            *float32 `faker:"oneof: 4.95, 9.99, 31997.97"`
	Price64            *float64 `faker:"oneof: 47463.9463525, 993747.95662529, 11131997.978767990"`
	NumS64             int64    `faker:"oneof: 1, 2"`
	PNumS64            *int64   `faker:"oneof: 1, 2"`
	NumS32             int32    `faker:"oneof: -3, 4"`
	PNumS32            *int32   `faker:"oneof: -3, 4"`
	NumS16             int16    `faker:"oneof: -5, 6"`
	PNumS16            *int16   `faker:"oneof: -5, 6"`
	NumS8              int8     `faker:"oneof: 7, -8"`
	PNumS8             *int8    `faker:"oneof: 7, -8"`
	NumU64             uint64   `faker:"oneof: 9, 10"`
	PNumU64            *uint64  `faker:"oneof: 9, 10"`
	NumU32             uint32   `faker:"oneof: 11, 12"`
	PNumU32            *uint32  `faker:"oneof: 11, 12"`
	NumU16             uint16   `faker:"oneof: 13, 14"`
	PNumU16            *uint16  `faker:"oneof: 13, 14"`
	NumU8              uint8    `faker:"oneof: 15, 16"`
	PNumU8             *uint8   `faker:"oneof: 15, 16"`
	NumU               uint     `faker:"oneof: 17, 18"`
	PNumU              *uint    `faker:"oneof: 17, 18"`
	Typ                string   `faker:"oneof: customer, customer_duplicate"`
	Bool1              bool     `faker:"bool"`
	PBool1             *bool    `faker:"bool"`
}

func TestGeneratedDataConversion(t *testing.T) {
	rand.Seed(rand.Int63())
	data := GenerateData{}
	faker.FakeData(&data)

	godata := FromGo(data)
	gonative := MustToGo(godata, WithOnlyTheseRules(WhenPath(nil, UseStruct(GenerateData{}))))

	assert.Equal(t, data, gonative)
}

type Max struct {
	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Float32 float32
	Float64 float64

	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
}

func TestMaxScalars(t *testing.T) {
	max := Max{
		Int:     math.MaxInt,
		Int8:    math.MaxInt8,
		Int16:   math.MaxInt16,
		Int32:   math.MaxInt32,
		Int64:   math.MaxInt64,
		Float32: math.MaxFloat32,
		Float64: math.MaxFloat64,
		Uint:    math.MaxUint,
		Uint8:   math.MaxInt8,
		Uint16:  math.MaxUint16,
		Uint32:  math.MaxUint32,
		Uint64:  math.MaxUint64,
	}

	t.Run("max scalars for respective values contain correct value", func(t *testing.T) {
		if runtime.GOARCH != "arm64" {
			t.Skip("skipping test that are for ARM64")
		}

		s := FromGo(max)
		r := MustToGo(s, WithOnlyTheseRules(WhenPath(nil, UseStruct(Max{}))))
		assert.Equal(t, max, r)
	})
	t.Run("test lossy conversion from Max float 64 to respective scalars", func(t *testing.T) {
		if runtime.GOARCH != "arm64" {
			t.Skip("skipping test that are for ARM64")
		}

		var m = Number(math.MaxFloat64)
		var s Schema = &Map{
			Field: []Field{
				{Name: "Int", Value: &m},
				{Name: "Int8", Value: &m},
				{Name: "Int16", Value: &m},
				{Name: "Int32", Value: &m},
				{Name: "Int64", Value: &m},
				{Name: "Float32", Value: &m},
				{Name: "Float64", Value: &m},
				{Name: "Uint", Value: &m},
				{Name: "Uint", Value: &m},
				{Name: "Uint8", Value: &m},
				{Name: "Uint16", Value: &m},
				{Name: "Uint32", Value: &m},
				{Name: "Uint64", Value: &m},
			},
		}
		r := MustToGo(s, WithOnlyTheseRules(WhenPath(nil, UseStruct(Max{})))).(Max)
		// Ints
		assert.Equal(t, int(math.Inf(1)), r.Int)
		assert.Equal(t, int8(math.Inf(1)), r.Int8)
		assert.Equal(t, int16(math.Inf(1)), r.Int16)
		// the fraction is discarded for ints
		assert.Equal(t, int32(-1), r.Int32)
		assert.Equal(t, int64(math.MaxInt64), r.Int64)
		// Floats
		assert.Equal(t, float32(math.Inf(1)), r.Float32)
		assert.Equal(t, math.MaxFloat64, r.Float64)
		// Uints
		assert.Equal(t, uint(math.Inf(1)), r.Uint)
		assert.Equal(t, uint8(math.Inf(1)), r.Uint8)
		assert.Equal(t, uint16(math.Inf(1)), r.Uint16)
		assert.Equal(t, uint32(math.Inf(1)), r.Uint32)
		assert.Equal(t, uint64(math.Inf(1)), r.Uint64)
	})
	t.Run("test lossy conversion from small float 64 to respective scalars", func(t *testing.T) {
		var m = Number(float64(3))
		var s Schema = &Map{
			Field: []Field{
				{Name: "Int", Value: &m},
				{Name: "Int8", Value: &m},
				{Name: "Int16", Value: &m},
				{Name: "Int32", Value: &m},
				{Name: "Int64", Value: &m},
				{Name: "Float32", Value: &m},
				{Name: "Float64", Value: &m},
				{Name: "Uint", Value: &m},
				{Name: "Uint", Value: &m},
				{Name: "Uint8", Value: &m},
				{Name: "Uint16", Value: &m},
				{Name: "Uint32", Value: &m},
				{Name: "Uint64", Value: &m},
			},
		}
		r := MustToGo(s, WithOnlyTheseRules(WhenPath(nil, UseStruct(Max{})))).(Max)
		// Ints
		assert.Equal(t, int(3), r.Int)
		assert.Equal(t, int8(3), r.Int8)
		assert.Equal(t, int16(3), r.Int16)
		// the fraction is discarded for ints
		assert.Equal(t, int32(3), r.Int32)
		assert.Equal(t, int64(3), r.Int64)
		// Floats
		assert.Equal(t, float32(3), r.Float32)
		assert.Equal(t, float64(3), r.Float64)
		// Uints
		assert.Equal(t, uint(3), r.Uint)
		assert.Equal(t, uint8(3), r.Uint8)
		assert.Equal(t, uint16(3), r.Uint16)
		assert.Equal(t, uint32(3), r.Uint32)
		assert.Equal(t, uint64(3), r.Uint64)
	})
}

func TestSchemaConversions(t *testing.T) {
	useCases := []struct {
		name string
		in   any
		out  Schema
		back any
	}{
		{
			name: "go list to schema and back",
			in:   []int{1, 2, 3},
			out: &List{
				Items: []Schema{
					MkInt(1),
					MkInt(2),
					MkInt(3),
				},
			},
			// Yes, back conversion always normalise to floats and []any
			// To map back to correct type use ToGo(_, WhenPath(nil, UseSlice(int)))
			back: []interface{}{
				float64(1),
				float64(2),
				float64(3),
			},
		},
		{
			name: "go list to schema and back",
			in: map[string]interface{}{
				"foo": 1,
				"bar": "str",
			},
			out: &Map{
				Field: []Field{
					{
						Name:  "foo",
						Value: MkInt(1),
					},
					{
						Name:  "bar",
						Value: MkString("str"),
					},
				},
			},
			back: map[string]interface{}{
				"foo": float64(1),
				"bar": "str",
			},
		},
	}
	for _, uc := range useCases {
		t.Run(uc.name, func(t *testing.T) {
			got := FromGo(uc.in)
			if assert.Equal(t, uc.out, got, "forward conversion issue") {
				assert.Equal(t, uc.back, MustToGo(got), "back conversion issue")
			}
		})
	}
}

type TestStruct1 struct {
	Foo   int
	Bar   string
	Other SharedStruct
}

type TestStruct2 struct {
	Baz   string
	Count int
}

type SharedStruct interface {
	shared()
}

var (
	_ SharedStruct = (*TestStruct1)(nil)
	_ SharedStruct = (*TestStruct2)(nil)
)

func (t *TestStruct1) shared() {}
func (t *TestStruct2) shared() {}

func TestSchemaToGoStructs(t *testing.T) {
	useCases := map[string]struct {
		in    Schema
		rules []RuleMatcher
		out   interface{}
	}{
		"schema struct to go struct": {
			in: &Map{
				Field: []Field{
					{
						Name:  "Foo",
						Value: MkInt(1),
					},
					{
						Name:  "Bar",
						Value: MkString("baz"),
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{}, UseStruct(TestStruct1{})),
			},
			out: TestStruct1{
				Foo: 1,
				Bar: "baz",
			},
		},
		"schema with list of structs": {
			in: &List{
				Items: []Schema{
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: MkInt(1),
							},
							{
								Name:  "Bar",
								Value: MkString("baz"),
							},
						},
					},
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: MkInt(13),
							},
							{
								Name:  "Bar",
								Value: MkString("baz3"),
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{"[*]"}, UseStruct(TestStruct1{})),
			},
			out: []any{
				TestStruct1{Foo: 1, Bar: "baz"},
				TestStruct1{Foo: 13, Bar: "baz3"},
			},
		},
		"struct with nested struct ": {
			in: &Map{
				Field: []Field{
					{
						Name:  "Foo",
						Value: MkInt(1),
					},
					{
						Name:  "Bar",
						Value: MkString("baz"),
					}, {
						Name: "Other",
						Value: &Map{
							Field: []Field{
								{
									Name:  "Count",
									Value: MkInt(41),
								},
								{
									Name:  "Baz",
									Value: MkString("baz2"),
								},
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{}, UseStruct(TestStruct1{})),
				WhenPath([]string{"Other"}, UseStruct(&TestStruct2{})),
			},
			out: TestStruct1{
				Foo: 1,
				Bar: "baz",
				Other: &TestStruct2{
					Baz:   "baz2",
					Count: 41,
				},
			},
		},
		"schema with list of structs with nested struct": {
			in: &List{
				Items: []Schema{
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: MkInt(1),
							},
							{
								Name:  "Bar",
								Value: MkString("baz"),
							},
							{
								Name: "Other",
								Value: &Map{
									Field: []Field{
										{
											Name:  "Baz",
											Value: MkString("baz2"),
										},
									},
								},
							},
						},
					},
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: MkInt(55),
							},
							{
								Name:  "Bar",
								Value: MkString("baz55"),
							},
							{
								Name: "Other",
								Value: &Map{
									Field: []Field{
										{
											Name:  "Foo",
											Value: MkInt(66),
										},
										{
											Name:  "Bar",
											Value: MkString("baz66"),
										},
									},
								},
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{"[*]"}, UseStruct(TestStruct1{})),
				WhenPath([]string{"[*]", "Other?.Foo"}, UseStruct(&TestStruct1{})),
				WhenPath([]string{"[*]", "Other?.Baz"}, UseStruct(&TestStruct2{})),
			},
			out: []any{
				TestStruct1{
					Foo: 1,
					Bar: "baz",
					Other: &TestStruct2{
						Baz: "baz2",
					},
				},
				TestStruct1{
					Foo: 55,
					Bar: "baz55",
					Other: &TestStruct1{
						Foo: 66,
						Bar: "baz66",
					},
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.out, MustToGo(uc.in, WithOnlyTheseRules(uc.rules...)))
		})
	}
}

type recordInTest[T any] struct {
	ID   string
	Data T
}

var (
	_ Unmarshaler = (*recordInTest[any])(nil)
	_ Marshaler   = (*recordInTest[any])(nil)
)

func (record *recordInTest[T]) MarshalSchema() (*Map, error) {
	var schemed Schema
	if _, ok := any(record.Data).(Schema); ok {
		schemed = any(record.Data).(Schema)
	} else {
		schemed = FromGo(record.Data)
	}

	return &Map{
		Field: []Field{
			{
				Name:  "ID",
				Value: MkString(record.ID),
			},
			{
				Name:  "Data",
				Value: schemed,
			},
		},
	}, nil
}

func (record *recordInTest[T]) UnmarshalSchema(x *Map) error {
	for _, field := range x.Field {
		switch field.Name {
		case "ID":
			if value, ok := As[string](field.Value); ok {
				record.ID = value
			}
		case "Data":
			data, err := ToGoG[T](field.Value)
			if err != nil {
				return fmt.Errorf(`recordInTest[T] BuildFromMapSchema: failed to convert "Data" value: %w`, err)
			}
			record.Data = data
		}
	}

	return nil
}

func TestSchemaMarshalSchema(t *testing.T) {
	useCases := map[string]struct {
		in  *recordInTest[Schema]
		out Schema
	}{
		"simple example of record": {
			in: &recordInTest[Schema]{
				ID: "foo",
				Data: MkMap(
					MkField("name", MkString("Alpha")),
					MkField("age", MkInt(42)),
				),
			},
			out: MkMap(
				MkField("ID", MkString("foo")),
				MkField("Data", MkMap(
					MkField("name", MkString("Alpha")),
					MkField("age", MkInt(42)),
				)),
			),
		},
	}

	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			result := FromGo(uc.in)
			assert.Equal(t, uc.out, result)

			out, err := ToGo(result, WithOnlyTheseRules(
				WhenPath([]string{}, UseStruct(&recordInTest[Schema]{})),
			))
			assert.NoError(t, err)
			assert.Equal(t, uc.in, out)
		})
	}

	t.Run("two conversions work on copy of a struct", func(t *testing.T) {
		r := WhenPath([]string{}, UseStruct(&recordInTest[Schema]{}))

		in1 := MkMap(
			MkField("ID", MkString("foo1")),
			MkField("Data", MkMap(
				MkField("name", MkString("Alpha1")),
				MkField("age", MkInt(41)),
			)),
		)
		in2 := MkMap(
			MkField("ID", MkString("foo2")),
			MkField("Data", MkMap(
				MkField("name", MkString("Alpha2")),
				MkField("age", MkInt(42)),
			)),
		)

		out1, err := ToGo(in1, WithOnlyTheseRules(r))
		assert.NoError(t, err)

		out2, err := ToGo(in2, WithOnlyTheseRules(r))
		assert.NoError(t, err)

		assert.NotEqual(t, out1, out2)
	})
}
