// Copyright 2019 ScyllaDB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package typedef

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/gocql/gocql"
	"golang.org/x/exp/rand"

	"github.com/scylladb/gemini/pkg/utils"
)

// nolint:revive
const (
	TYPE_UDT   = "udt"
	TYPE_MAP   = "map"
	TYPE_LIST  = "list"
	TYPE_SET   = "set"
	TYPE_TUPLE = "tuple"
)

// nolint:revive
const (
	TYPE_ASCII     = SimpleType("ascii")
	TYPE_BIGINT    = SimpleType("bigint")
	TYPE_BLOB      = SimpleType("blob")
	TYPE_BOOLEAN   = SimpleType("boolean")
	TYPE_DATE      = SimpleType("date")
	TYPE_DECIMAL   = SimpleType("decimal")
	TYPE_DOUBLE    = SimpleType("double")
	TYPE_DURATION  = SimpleType("duration")
	TYPE_FLOAT     = SimpleType("float")
	TYPE_INET      = SimpleType("inet")
	TYPE_INT       = SimpleType("int")
	TYPE_SMALLINT  = SimpleType("smallint")
	TYPE_TEXT      = SimpleType("text")
	TYPE_TIME      = SimpleType("time")
	TYPE_TIMESTAMP = SimpleType("timestamp")
	TYPE_TIMEUUID  = SimpleType("timeuuid")
	TYPE_TINYINT   = SimpleType("tinyint")
	TYPE_UUID      = SimpleType("uuid")
	TYPE_VARCHAR   = SimpleType("varchar")
	TYPE_VARINT    = SimpleType("varint")
)

const (
	maxMapSize = 10
	maxBagSize = 10
)

var (
	TypesMapKeyBlacklist = map[SimpleType]struct{}{
		TYPE_BLOB:     {},
		TYPE_DURATION: {},
	}
	TypesForIndex     = SimpleTypes{TYPE_DECIMAL, TYPE_DOUBLE, TYPE_FLOAT, TYPE_INT, TYPE_SMALLINT, TYPE_TINYINT, TYPE_VARINT}
	PartitionKeyTypes = SimpleTypes{
		TYPE_ASCII, TYPE_BIGINT, TYPE_DATE, TYPE_DECIMAL, TYPE_DOUBLE,
		TYPE_FLOAT, TYPE_INET, TYPE_INT, TYPE_SMALLINT, TYPE_TEXT, TYPE_TIME, TYPE_TIMESTAMP, TYPE_TIMEUUID,
		TYPE_TINYINT, TYPE_UUID, TYPE_VARCHAR, TYPE_VARINT, TYPE_BOOLEAN,
	}

	PkTypes = SimpleTypes{
		TYPE_ASCII, TYPE_BIGINT, TYPE_BLOB, TYPE_DATE, TYPE_DECIMAL, TYPE_DOUBLE,
		TYPE_FLOAT, TYPE_INET, TYPE_INT, TYPE_SMALLINT, TYPE_TEXT, TYPE_TIME, TYPE_TIMESTAMP, TYPE_TIMEUUID,
		TYPE_TINYINT, TYPE_UUID, TYPE_VARCHAR, TYPE_VARINT,
	}
	AllTypes = append(append(SimpleTypes{}, PkTypes...), TYPE_BOOLEAN, TYPE_DURATION)
)

var goCQLTypeMap = map[gocql.Type]gocql.TypeInfo{
	gocql.TypeAscii:     gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeAscii, ""),
	gocql.TypeBigInt:    gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeBigInt, ""),
	gocql.TypeBlob:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeBlob, ""),
	gocql.TypeBoolean:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeBoolean, ""),
	gocql.TypeDate:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeDate, ""),
	gocql.TypeDecimal:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeDecimal, ""),
	gocql.TypeDouble:    gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeDouble, ""),
	gocql.TypeDuration:  gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeDuration, ""),
	gocql.TypeFloat:     gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeFloat, ""),
	gocql.TypeInet:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeInet, ""),
	gocql.TypeInt:       gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeInt, ""),
	gocql.TypeSmallInt:  gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeSmallInt, ""),
	gocql.TypeText:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeText, ""),
	gocql.TypeTime:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeTime, ""),
	gocql.TypeTimestamp: gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeTimestamp, ""),
	gocql.TypeTimeUUID:  gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeTimeUUID, ""),
	gocql.TypeTinyInt:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeTinyInt, ""),
	gocql.TypeUUID:      gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeUUID, ""),
	gocql.TypeVarchar:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeVarchar, ""),
	gocql.TypeVarint:    gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeVarint, ""),

	// Complex coltypes
	gocql.TypeList:  gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeList, ""),
	gocql.TypeMap:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeMap, ""),
	gocql.TypeSet:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeSet, ""),
	gocql.TypeTuple: gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeTuple, ""),
	gocql.TypeUDT:   gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeUDT, ""),

	// Special coltypes
	gocql.TypeCounter: gocql.NewNativeType(GoCQLProtoVersion4, gocql.TypeCounter, ""),
}

func GetGoCQLTypeMap() map[gocql.Type]gocql.TypeInfo {
	return goCQLTypeMap
}

type MapType struct {
	ComplexType string     `json:"complex_type"`
	KeyType     SimpleType `json:"key_type"`
	ValueType   SimpleType `json:"value_type"`
	Frozen      bool       `json:"frozen"`
}

func (mt *MapType) CQLType() gocql.TypeInfo {
	return goCQLTypeMap[gocql.TypeMap]
}

func (mt *MapType) Name() string {
	if mt.Frozen {
		return "frozen<map<" + mt.KeyType.Name() + "," + mt.ValueType.Name() + ">>"
	}
	return "map<" + mt.KeyType.Name() + "," + mt.ValueType.Name() + ">"
}

func (mt *MapType) CQLHolder() string {
	return "?"
}

func (mt *MapType) CQLPretty(value interface{}) string {
	if reflect.TypeOf(value).Kind() != reflect.Map {
		panic(fmt.Sprintf("map cql pretty, unknown type %v", mt))
	}
	vof := reflect.ValueOf(value)
	s := vof.MapRange()
	out := make([]string, len(vof.MapKeys()))
	id := 0
	for s.Next() {
		out[id] = fmt.Sprintf("%s:%s", mt.KeyType.CQLPretty(s.Key().Interface()), mt.ValueType.CQLPretty(s.Value().Interface()))
		id++
	}
	return fmt.Sprintf("{%s}", strings.Join(out, ","))
}

func (mt *MapType) GenJSONValue(r *rand.Rand, p *PartitionRangeConfig) interface{} {
	count := r.Intn(9) + 1
	vals := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(mt.KeyType.GenJSONValue(r, p)), reflect.TypeOf(mt.ValueType.GenJSONValue(r, p))))
	for i := 0; i < count; i++ {
		vals.SetMapIndex(reflect.ValueOf(mt.KeyType.GenJSONValue(r, p)), reflect.ValueOf(mt.ValueType.GenJSONValue(r, p)))
	}
	return vals.Interface()
}

func (mt *MapType) GenValue(r *rand.Rand, p *PartitionRangeConfig) []interface{} {
	count := utils.RandInt2(r, 1, maxMapSize+1)
	vals := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(mt.KeyType.GenValue(r, p)[0]), reflect.TypeOf(mt.ValueType.GenValue(r, p)[0])))
	for i := 0; i < count; i++ {
		vals.SetMapIndex(reflect.ValueOf(mt.KeyType.GenValue(r, p)[0]), reflect.ValueOf(mt.ValueType.GenValue(r, p)[0]))
	}
	return []interface{}{vals.Interface()}
}

func (mt *MapType) LenValue() int {
	return 1
}

func (mt *MapType) CQLDef() string {
	if mt.Frozen {
		return "frozen<map<" + mt.KeyType.CQLDef() + "," + mt.ValueType.CQLDef() + ">>"
	}
	return "map<" + mt.KeyType.CQLDef() + "," + mt.ValueType.CQLDef() + ">"
}

func (mt *MapType) Indexable() bool {
	return false
}

// ValueVariationsNumber returns number of bytes generated value holds
func (mt *MapType) ValueVariationsNumber(p *PartitionRangeConfig) float64 {
	return math.Pow(mt.KeyType.ValueVariationsNumber(p)*mt.ValueType.ValueVariationsNumber(p), maxMapSize)
}

type CounterType struct {
	Value int64
}

func (ct *CounterType) CQLType() gocql.TypeInfo {
	return goCQLTypeMap[gocql.TypeMap]
}

func (ct *CounterType) Name() string {
	return "counter"
}

func (ct *CounterType) CQLHolder() string {
	return "?"
}

func (ct *CounterType) CQLPretty(value interface{}) string {
	return fmt.Sprintf("%d", value)
}

func (ct *CounterType) GenJSONValue(r *rand.Rand, _ *PartitionRangeConfig) interface{} {
	if utils.UnderTest {
		return r.Int63()
	}
	return atomic.AddInt64(&ct.Value, 1)
}

func (ct *CounterType) GenValue(r *rand.Rand, _ *PartitionRangeConfig) []interface{} {
	if utils.UnderTest {
		return []interface{}{r.Int63()}
	}
	return []interface{}{atomic.AddInt64(&ct.Value, 1)}
}

func (ct *CounterType) LenValue() int {
	return 1
}

func (ct *CounterType) CQLDef() string {
	return "counter"
}

func (ct *CounterType) Indexable() bool {
	return false
}

// ValueVariationsNumber returns number of bytes generated value holds
func (ct *CounterType) ValueVariationsNumber(_ *PartitionRangeConfig) float64 {
	// As a type, counters are a 64-bit signed integer
	return 2 ^ 64
}
