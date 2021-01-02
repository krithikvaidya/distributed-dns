package main


import "fmt"

type node struct {
	next *node
	key  string
	data string
}

type linkedlist struct {
	head *node
	size int
}

//Creates a new instance of linkedlist
func newLinkedList() *linkedlist {
	return &linkedlist{
		head: nil,
		size: 0,
	}
}

//Check is list is empty
func (ll *linkedlist) isEmpty() bool {
	if ll.head == nil {
		return true
	}
	return false
}

//Adds a new pair to linkedlist
func (ll *linkedlist) add(strA, strB string) {
	newNode := &node{
		next: nil,
		key:  strA,
		data: strB,
	}
	if ll.head != nil {
		newNode.next = ll.head
	}
	ll.head = newNode
	ll.size++
}

//Removes the corresponding key value pair
func (ll *linkedlist) remove(str string) bool {
	var newNode = ll.head
	if newNode.key == str {
		ll.head = ll.head.next
		ll.size--
		return true
	}
	for {
		if newNode.next == nil {
			break
		}
		if newNode.next.key == str {
			newNode.next = newNode.next.next
			ll.size--
			return true
		}
		newNode = newNode.next
	}
	return false
}

//To view the contents of a bucket
func (ll *linkedlist) printlist() {
	var newNode = ll.head
	for {
		if newNode == nil {
			break
		}
		fmt.Printf("%s,%s -> ", newNode.key, newNode.data)
		newNode = newNode.next
	}
	fmt.Print("\n")
}
