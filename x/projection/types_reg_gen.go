// Code generated by mkunion. DO NOT EDIT.
package projection

import (
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
	"testing"
)

func init() {
	shared.TypeRegistryStore[predicate.WherePredicates]("github.com/widmogrod/mkunion/x/storage/predicate.WherePredicates")
	shared.TypeRegistryStore[AtWatermark]("github.com/widmogrod/mkunion/x/projection.AtWatermark")
	shared.TypeRegistryStore[Data[Either[int, float64]]]("github.com/widmogrod/mkunion/x/projection.Data[Either[int,float64]]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Data[Either[int,float64]]", DataFromJSON[Either[int, float64]], DataToJSON[Either[int, float64]])
	shared.TypeRegistryStore[Data[any]]("github.com/widmogrod/mkunion/x/projection.Data[any]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Data[any]", DataFromJSON[any], DataToJSON[any])
	shared.TypeRegistryStore[Data[float64]]("github.com/widmogrod/mkunion/x/projection.Data[float64]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Data[float64]", DataFromJSON[float64], DataToJSON[float64])
	shared.TypeRegistryStore[Data[int]]("github.com/widmogrod/mkunion/x/projection.Data[int]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Data[int]", DataFromJSON[int], DataToJSON[int])
	shared.TypeRegistryStore[Data[string]]("github.com/widmogrod/mkunion/x/projection.Data[string]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Data[string]", DataFromJSON[string], DataToJSON[string])
	shared.TypeRegistryStore[Discard]("github.com/widmogrod/mkunion/x/projection.Discard")
	shared.TypeRegistryStore[Either[*Record[int], *Record[float64]]]("github.com/widmogrod/mkunion/x/projection.Either[*Record[int],*Record[float64]]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Either[*Record[int],*Record[float64]]", EitherFromJSON[*Record[int], *Record[float64]], EitherToJSON[*Record[int], *Record[float64]])
	shared.TypeRegistryStore[Either[any, any]]("github.com/widmogrod/mkunion/x/projection.Either[any,any]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Either[any,any]", EitherFromJSON[any, any], EitherToJSON[any, any])
	shared.TypeRegistryStore[Either[int, float64]]("github.com/widmogrod/mkunion/x/projection.Either[int,float64]")
	shared.JSONMarshallerRegister("github.com/widmogrod/mkunion/x/projection.Either[int,float64]", EitherFromJSON[int, float64], EitherToJSON[int, float64])
	shared.TypeRegistryStore[FixedWindow]("github.com/widmogrod/mkunion/x/projection.FixedWindow")
	shared.TypeRegistryStore[JoinContextState]("github.com/widmogrod/mkunion/x/projection.JoinContextState")
	shared.TypeRegistryStore[Left[*Record[int], *Record[float64]]]("github.com/widmogrod/mkunion/x/projection.Left[*Record[int],*Record[float64]]")
	shared.TypeRegistryStore[Left[any, any]]("github.com/widmogrod/mkunion/x/projection.Left[any,any]")
	shared.TypeRegistryStore[Left[int, float64]]("github.com/widmogrod/mkunion/x/projection.Left[int,float64]")
	shared.TypeRegistryStore[PullPushContextState]("github.com/widmogrod/mkunion/x/projection.PullPushContextState")
	shared.TypeRegistryStore[PushAndPullInMemoryContext[any, int]]("github.com/widmogrod/mkunion/x/projection.PushAndPullInMemoryContext[any,int]")
	shared.TypeRegistryStore[PushAndPullInMemoryContext[float64, float64]]("github.com/widmogrod/mkunion/x/projection.PushAndPullInMemoryContext[float64,float64]")
	shared.TypeRegistryStore[PushAndPullInMemoryContext[int, float64]]("github.com/widmogrod/mkunion/x/projection.PushAndPullInMemoryContext[int,float64]")
	shared.TypeRegistryStore[PushAndPull[int, int]]("github.com/widmogrod/mkunion/x/projection.PushAndPull[int,int]")
	shared.TypeRegistryStore[Record[Either[int, float64]]]("github.com/widmogrod/mkunion/x/projection.Record[Either[int,float64]]")
	shared.TypeRegistryStore[Record[any]]("github.com/widmogrod/mkunion/x/projection.Record[any]")
	shared.TypeRegistryStore[Record[float64]]("github.com/widmogrod/mkunion/x/projection.Record[float64]")
	shared.TypeRegistryStore[Record[int]]("github.com/widmogrod/mkunion/x/projection.Record[int]")
	shared.TypeRegistryStore[Record[string]]("github.com/widmogrod/mkunion/x/projection.Record[string]")
	shared.TypeRegistryStore[RecoveryOptions[SnapshotState]]("github.com/widmogrod/mkunion/x/projection.RecoveryOptions[SnapshotState]")
	shared.TypeRegistryStore[Right[*Record[int], *Record[float64]]]("github.com/widmogrod/mkunion/x/projection.Right[*Record[int],*Record[float64]]")
	shared.TypeRegistryStore[Right[any, any]]("github.com/widmogrod/mkunion/x/projection.Right[any,any]")
	shared.TypeRegistryStore[Right[int, float64]]("github.com/widmogrod/mkunion/x/projection.Right[int,float64]")
	shared.TypeRegistryStore[SessionWindow]("github.com/widmogrod/mkunion/x/projection.SessionWindow")
	shared.TypeRegistryStore[SimulateProblem]("github.com/widmogrod/mkunion/x/projection.SimulateProblem")
	shared.TypeRegistryStore[SlidingWindow]("github.com/widmogrod/mkunion/x/projection.SlidingWindow")
	shared.TypeRegistryStore[Watermark[Either[int, float64]]]("github.com/widmogrod/mkunion/x/projection.Watermark[Either[int,float64]]")
	shared.TypeRegistryStore[Watermark[any]]("github.com/widmogrod/mkunion/x/projection.Watermark[any]")
	shared.TypeRegistryStore[Watermark[float64]]("github.com/widmogrod/mkunion/x/projection.Watermark[float64]")
	shared.TypeRegistryStore[Watermark[int]]("github.com/widmogrod/mkunion/x/projection.Watermark[int]")
	shared.TypeRegistryStore[Watermark[string]]("github.com/widmogrod/mkunion/x/projection.Watermark[string]")
	shared.TypeRegistryStore[Window]("github.com/widmogrod/mkunion/x/projection.Window")
	shared.TypeRegistryStore[schemaless.FindingRecords[schemaless.Record[SnapshotState]]]("github.com/widmogrod/mkunion/x/storage/schemaless.FindingRecords[Record[github.com/widmogrod/mkunion/x/projection.SnapshotState]]")
	shared.TypeRegistryStore[schemaless.Repository[SnapshotState]]("github.com/widmogrod/mkunion/x/storage/schemaless.Repository[github.com/widmogrod/mkunion/x/projection.SnapshotState]")
	shared.TypeRegistryStore[stream.EventTime]("github.com/widmogrod/mkunion/x/stream.EventTime")
	shared.TypeRegistryStore[stream.Item[schema.Schema]]("github.com/widmogrod/mkunion/x/stream.Item[github.com/widmogrod/mkunion/x/schema.Schema]")
	shared.TypeRegistryStore[stream.Offset]("github.com/widmogrod/mkunion/x/stream.Offset")
	shared.TypeRegistryStore[stream.Stream[schema.Schema]]("github.com/widmogrod/mkunion/x/stream.Stream[github.com/widmogrod/mkunion/x/schema.Schema]")
	shared.TypeRegistryStore[testing.T]("testing.T")
}
