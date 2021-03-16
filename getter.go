package config

import "fmt"

func Must(repo *Repository, key string) Value {
	v, ok := repo.Get(NewKey(key))
	if !ok {
		panic(fmt.Sprintf("Unregistered config key: %q", key))
	}
	return v
}

func MustStr(repo *Repository, key string) string {
	return Must(repo, key).(string)
}

func MustInt(repo *Repository, key string) int {
	return Must(repo, key).(int)
}

func MustInt8(repo *Repository, key string) int8 {
	return Must(repo, key).(int8)
}

func MustInt16(repo *Repository, key string) int16 {
	return Must(repo, key).(int16)
}

func MustInt32(repo *Repository, key string) int32 {
	return Must(repo, key).(int32)
}

func MustInt64(repo *Repository, key string) int64 {
	return Must(repo, key).(int64)
}

func MustUint(repo *Repository, key string) uint {
	return Must(repo, key).(uint)
}

func MustUint8(repo *Repository, key string) uint8 {
	return Must(repo, key).(uint8)
}

func MustUint16(repo *Repository, key string) uint16 {
	return Must(repo, key).(uint16)
}

func MustUint32(repo *Repository, key string) uint32 {
	return Must(repo, key).(uint32)
}

func MustUint64(repo *Repository, key string) uint64 {
	return Must(repo, key).(uint64)
}

func MustUintptr(repo *Repository, key string) uintptr {
	return Must(repo, key).(uintptr)
}

func MustBool(repo *Repository, key string) bool {
	return Must(repo, key).(bool)
}

func MustFloat32(repo *Repository, key string) float32 {
	return Must(repo, key).(float32)
}

func MustFloat64(repo *Repository, key string) float64 {
	return Must(repo, key).(float64)
}

func MustStrArr(repo *Repository, key string) []string {
	return Must(repo, key).([]string)
}

func MustIntArr(repo *Repository, key string) []int {
	return Must(repo, key).([]int)
}
