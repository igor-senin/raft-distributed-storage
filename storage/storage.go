package storage

type (
	TKey   int
	TValue int
)

var storage map[TKey]TValue

func InitStorage() {
	storage = make(map[TKey]TValue)
}

func ReadRecord(key TKey) (TValue, bool) {
	value, ok := storage[key]
	return value, ok
}

func CreateRecord(key TKey, value TValue) error {
	storage[key] = value
	return nil
}

func UpdateRecord(key TKey, value TValue) (TValue, bool) {
	old_value, ok := storage[key]
	storage[key] = value
	return old_value, ok
}

func DeleteRecord(key TKey) (TValue, bool) {
	defer delete(storage, key)
	value, ok := storage[key]
	return value, ok
}
