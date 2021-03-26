package kv_store

import "fmt"

type Node struct {
	Next *Node
	Key  string
	Data string
}

type Linkedlist struct {
	Head *Node
	Size int
}

//Creates a new instance of linkedlist
func newLinkedList() *Linkedlist {
	return &Linkedlist{
		Head: nil,
		Size: 0,
	}
}

//Check if list is empty
func (ll *Linkedlist) isEmpty() bool {
	if ll.Head == nil {
		return true
	}
	return false
}

//Adds a new pair to linkedlist
func (ll *Linkedlist) add(strA, strB string) {
	newNode := &Node{
		Next: nil,
		Key:  strA,
		Data: strB,
	}
	if ll.Head != nil {
		newNode.Next = ll.Head
	}
	ll.Head = newNode
	ll.Size++
}

//Removes the corresponding key value pair
func (ll *Linkedlist) remove(str string) bool {
	var newNode = ll.Head
	if ll.Head == nil {
		return false
	}

	if newNode.Key == str {
		ll.Head = ll.Head.Next
		ll.Size--
		return true
	}
	for {
		if newNode.Next == nil {
			break
		}
		if newNode.Next.Key == str {
			newNode.Next = newNode.Next.Next
			ll.Size--
			return true
		}
		newNode = newNode.Next
	}
	return false
}

//To view the contents of a bucket
func (ll *Linkedlist) printlist() {
	var newNode = ll.Head
	for {
		if newNode == nil {
			break
		}
		fmt.Printf("%s,%s -> ", newNode.Key, newNode.Data)
		newNode = newNode.Next
	}
	fmt.Print("\n")
}
