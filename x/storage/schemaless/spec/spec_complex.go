package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec/specdata"
)

// NewComplexRepoFunc mirrors NewRepoFunc for the complex suite, which stores
// records whose payload nests a union (specdata.Vehicle).
type NewComplexRepoFunc func(t *testing.T) schemaless.Repository[specdata.Vehicle]

func seedVehicles(recordType string) []schemaless.Record[specdata.Vehicle] {
	return []schemaless.Record[specdata.Vehicle]{
		{ID: "beetle", Type: recordType, Data: specdata.Vehicle{
			Name: "beetle", Wheels: 4,
			Engine: &specdata.Petrol{Brand: "vw", Cylinders: 4},
			Tags:   []string{"classic"},
		}},
		{ID: "model3", Type: recordType, Data: specdata.Vehicle{
			Name: "model3", Wheels: 4,
			Engine: &specdata.Electric{Brand: "tesla", KWh: 75},
			Tags:   []string{"ev", "fast"},
		}},
		{ID: "hauler", Type: recordType, Data: specdata.Vehicle{
			Name: "hauler", Wheels: 6,
			Engine: &specdata.Petrol{Brand: "volvo", Cylinders: 8},
			Tags:   []string{"work"},
		}},
		{ID: "cargo", Type: recordType, Data: specdata.Vehicle{
			Name: "cargo", Wheels: 2,
			Engine: &specdata.Electric{Brand: "bosch", KWh: 0.5},
			Tags:   []string{"ev"},
		}},
	}
}

func mustSaveVehicles(t *testing.T, repo schemaless.Repository[specdata.Vehicle], records ...schemaless.Record[specdata.Vehicle]) {
	t.Helper()
	result, err := repo.UpdateRecords(schemaless.Save(records...))
	require.NoError(t, err, "seeding vehicle records must succeed")
	require.Len(t, result.Saved, len(records))
}

func findAllVehicles(t *testing.T, repo schemaless.Repository[specdata.Vehicle], query schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]) []schemaless.Record[specdata.Vehicle] {
	t.Helper()
	var items []schemaless.Record[specdata.Vehicle]
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

func vehicleIDs(items []schemaless.Record[specdata.Vehicle]) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.ID
	}
	return result
}

// RunComplexQuerySpec runs the specification for records whose payload nests
// a union, using union-path locations and composed AND/OR/NOT predicates.
// Like RunRepositorySpec, downgraded behaviours are reported as skipped and
// the run contributes to the backend's capability report.
func RunComplexQuerySpec(t *testing.T, backend string, newRepo NewComplexRepoFunc, caps Capabilities) {
	r := newRunner(t, backend, suiteComplex, caps)

	r.run("union variants survive a round trip", func(t *testing.T) {
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

	r.run("filter on a field inside a union variant", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["specdata.Petrol"].Cylinders >= :cylinders`,
				predicate.ParamBinds{":cylinders": schema.MkInt(8)},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"hauler"}, vehicleIDs(items))
	})

	r.run("OR matches across union variants", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["specdata.Petrol"].Brand = :petrolBrand OR Data.Engine["specdata.Electric"].Brand = :electricBrand`,
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

	r.run("NOT excludes a union variant match, records without the field stay", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`NOT Data.Engine["specdata.Electric"].Brand = :brand`,
				predicate.ParamBinds{":brand": schema.MkString("tesla")},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle", "hauler", "cargo"}, vehicleIDs(items),
			"petrol vehicles have no Electric branch, so NOT must keep them")
	})

	r.run("AND combines a plain field with a union field", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Wheels = :wheels AND Data.Engine["specdata.Electric"].KWh > :kwh`,
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

	r.run("numeric equality filter", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
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

	r.run("string equality filter", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
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

	r.run("filter on a nonexistent field returns no records and no error", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
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

	r.run("update of a union variant is queryable afterwards", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSaveVehicles(t, repo, seedVehicles(recordType)...)

		// convert the beetle to an electric engine
		beetle, err := repo.Get("beetle", recordType)
		require.NoError(t, err)
		beetle.Data.Engine = &specdata.Electric{Brand: "retrofit", KWh: 40}
		mustSaveVehicles(t, repo, beetle)

		items := findAllVehicles(t, repo, schemaless.FindingRecords[schemaless.Record[specdata.Vehicle]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				`Data.Engine["specdata.Electric"].Brand = :brand`,
				predicate.ParamBinds{":brand": schema.MkString("retrofit")},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"beetle"}, vehicleIDs(items))
	})
}
