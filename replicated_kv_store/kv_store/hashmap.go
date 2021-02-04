package kv_store

const (
	prime = 17003
)

//Hash function to convert string to an array index value
func hash(str string) int {
	var res = 1
	for i := 0; i < len(str); i++ {
		x := int(str[i] - 'A')
		if x < 0 {
			x = -1 * x
		}
		res += (res*prime + x)
		res = res % 101
	}
	//fmt.Println(res)
	return res
}

//Push is to create new key
func (kv *store) Push(key, value string) {
	id := hash(key)
	if kv.db[id] == nil {
		kv.db[id] = newLinkedList()
	}
	kv.db[id].add(key, value)
}

//Get is to return key
func (kv *store) Get(key string) string {
	id := hash(key)
	if kv.db[id] == nil {
		return "Invalid"
	}
	newNode := kv.db[id].Head
	for {
		if newNode == nil {
			break
		} else if newNode.Key == key {
			return newNode.Data
		}
		newNode = newNode.Next
	}
	return "Invalid"
}

//Put is to update key
func (kv *store) Put(key, value string) bool {
	id := hash(key)
	if kv.db[id] == nil {
		return false
	}
	newNode := kv.db[id].Head
	for {
		if newNode == nil {
			break
		} else if newNode.Key == key {
			newNode.Data = value
			return true
		}
		newNode = newNode.Next
	}
	return false
}

//Delete the key
func (kv *store) Delete(key string) bool {
	id := hash(key)
	if kv.db[id] == nil {
		return false
	}
	check := kv.db[id].remove(key)
	return check
}
