package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// Engine is a union stored inside the record payload, so the complex suite
// exercises nested-union serialisation and union-path queries on every backend.
//
//go:tag mkunion:"Engine"
type (
	Petrol struct {
		Brand     string
		Cylinders int
	}
	Electric struct {
		Brand string
		KWh   float64
	}
)

// Vehicle is the "complex" record payload: plain fields, a nested union,
// and a list.
//
//go:tag serde:"json"
type Vehicle struct {
	Name   string
	Wheels int
	Engine Engine
	Tags   []string
}

type (
	// ComplexRepo stores records whose payload nests a union.
	ComplexRepo = schemaless.Repository[Vehicle]
	// NewComplexRepoFunc mirrors NewRepoFunc for the complex suite.
	NewComplexRepoFunc func(t *testing.T) ComplexRepo
)

func seedVehicles(recordType string) []schemaless.Record[Vehicle] {
	return []schemaless.Record[Vehicle]{
		{ID: "beetle", Type: recordType, Data: Vehicle{
			Name: "beetle", Wheels: 4,
			Engine: &Petrol{Brand: "vw", Cylinders: 4},
			Tags:   []string{"classic"},
		}},
		{ID: "model3", Type: recordType, Data: Vehicle{
			Name: "model3", Wheels: 4,
			Engine: &Electric{Brand: "tesla", KWh: 75},
			Tags:   []string{"ev", "fast"},
		}},
		{ID: "hauler", Type: recordType, Data: Vehicle{
			Name: "hauler", Wheels: 6,
			Engine: &Petrol{Brand: "volvo", Cylinders: 8},
			Tags:   []string{"work"},
		}},
		{ID: "cargo", Type: recordType, Data: Vehicle{
			Name: "cargo", Wheels: 2,
			Engine: &Electric{Brand: "bosch", KWh: 0.5},
			Tags:   []string{"ev"},
		}},
	}
}

func mustSaveVehicles(t *testing.T, repo ComplexRepo, records ...schemaless.Record[Vehicle]) {
	t.Helper()
	result, err := repo.UpdateRecords(schemaless.Save(records...))
	require.NoError(t, err, "seeding vehicle records must succeed")
	require.Len(t, result.Saved, len(records))
}

func findAllVehicles(t *testing.T, repo ComplexRepo, query schemaless.FindingRecords[schemaless.Record[Vehicle]]) []schemaless.Record[Vehicle] {
	t.Helper()
	var items []schemaless.Record[Vehicle]
	for i := 0; ; i++ {
		require.Less(t, i, 100, "pagination must terminate")
		page, err := repo.FindingRecords(query)
		require.NoError(t, err, "finding records must succeed")
		items = append(items, page.Items...)
		if !page.HasNext() {
			return items
		}
		query = *page.Next
	}
}

func vehicleIDs(items []schemaless.Record[Vehicle]) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.ID
	}
	return result
}

// RunComplexQuerySpec runs the specification for records whose payload nests
// a union, using union-path locations and composed AND/OR/NOT predicates.
// Like RunRepositorySpec, downgraded behaviours are reported as skipped.
func RunComplexQuerySpec(t *testing.T, newRepo NewComplexRepoFunc, caps Capabilities) {
	t.Run("union variants survive a round trip", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		seeded := seedVehicles(recordType)
		mustSaveVehicles(t, repo, seeded...)

		for _, want := range seeded {
			got, err := repo.Get(want.ID, recordType)
			require.NoError(t, err)
			assert.Equal(t, want.Data, got.Data, "payload with nested union must round-trip unchanged")
		}
	})

	t.Run("filter on a field inside a union variant", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["spec.Petrol"].Cylinders >= :cylinders`,
				predicate.ParamBinds{":cylinders": schema.MkInt(8)},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"hauler"}, vehicleIDs(items))
	})

	t.Run("OR matches across union variants", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["spec.Petrol"].Brand = :petrolBrand OR Data.Engine["spec.Electric"].Brand = :electricBrand`,
				predicate.ParamBinds{
					":petrolBrand":   schema.MkString("vw"),
					":electricBrand": schema.MkString("tesla"),
				},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle", "model3"}, vehicleIDs(items))
	})

	t.Run("NOT excludes a union variant match, records without the field stay", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`NOT Data.Engine["spec.Electric"].Brand = :brand`,
				predicate.ParamBinds{":brand": schema.MkString("tesla")},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle", "hauler", "cargo"}, vehicleIDs(items),
			"petrol vehicles have no Electric branch, so NOT must keep them")
	})

	t.Run("AND combines a plain field with a union field", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Wheels = :wheels AND Data.Engine["spec.Electric"].KWh > :kwh`,
				predicate.ParamBinds{
					":wheels": schema.MkInt(4),
					":kwh":    schema.MkFloat(1),
				},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"model3"}, vehicleIDs(items))
	})

	t.Run("numeric equality filter", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Wheels = :wheels`,
				predicate.ParamBinds{":wheels": schema.MkInt(4)},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle", "model3"}, vehicleIDs(items))
	})

	t.Run("string equality filter", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Name = :name`,
				predicate.ParamBinds{":name": schema.MkString("hauler")},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"hauler"}, vehicleIDs(items))
	})

	t.Run("filter on a nonexistent field returns no records and no error", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.NoSuchField = :value`,
				predicate.ParamBinds{":value": schema.MkString("anything")},
				nil,
			),
			Limit: 10,
		})
		assert.Empty(t, items, "an unknown field matches nothing, it is not an error")
	})

	t.Run("update of a union variant is queryable afterwards", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		// convert the beetle to an electric engine
		beetle, err := repo.Get("beetle", recordType)
		require.NoError(t, err)
		beetle.Data.Engine = &Electric{Brand: "retrofit", KWh: 40}
		mustSaveVehicles(t, repo, beetle)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["spec.Electric"].Brand = :brand`,
				predicate.ParamBinds{":brand": schema.MkString("retrofit")},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle"}, vehicleIDs(items))
	})
}
