package iteraTraversal

import (
	"container/list"
)

type TreeNode struct {
	Value int
	Left  *TreeNode
	Right *TreeNode
}

func preorderTraversal(root *TreeNode) []int {
	ans := []int{}

	if root == nil {
		return ans
	}

	st := list.New()  //创建一个链表
	st.PushBack(root) //将一个值为root的新元素插入链表的最后一个位置，返回生成的新元素

	for st.Len() > 0 {
		node := st.Remove(st.Back()).(*TreeNode) //back方法返回链表最后一个元素，Remove方法删除链表中的元素,这里就是链表最后一个元素

		ans = append(ans, node.Value)
		if node.Right != nil {
			st.PushBack(node.Right)
		}
		if node.Left != nil {
			st.PushBack(node.Left)
		}
	}
	return ans
}

func inorderTraversal(root *TreeNode) []int {
	ans := []int{}
	if root == nil {
		return ans
	}
	st := list.New()
	cur := root
	for cur != nil || st.Len() > 0 {
		if cur != nil {
			st.PushBack(root)
			cur = cur.Left
		} else {
			node := st.Remove(st.Back()).(*TreeNode)
			ans = append(ans, node.Value)
			cur = cur.Right
		}
	}
	return ans
}

//后序遍历
func postorderTraverasl(root *TreeNode) []int{
	ans := []int{}
	if root != nil {
		return ans
	}
	//定义一个链表
	st := list.New()
	st.PushBack(root)
	for (st.Len() > 0){
		node := st.Remove(st.Back()).(*TreeNode)
		ans = append(ans, node.Value)
		if node.Left != nil{
			st.PushBack(node.Left)
		}
		if node.Right != nil {
			st.PushBack(node.Right)
		}
	}
	reverse(ans)
	return ans
}

func reverse(arr []int) {
	for i,j := 0, len(arr)-1; i < j; i++{
		arr[i], arr[j] = arr[j], arr[i]
		j--
	}
}

//层序遍历
func levelOrder(root *TreeNode) [][]int{
	res := [][]int{}
	temp := []int{}
	st := list.New()
	st.PushBack(root)
	for st.Len() > 0{
		length := st.Len()  //定义每一层所含有的元素个数
		for i := 0; i < length; i++{
			node := st.Remove(st.Front()).(*TreeNode)
			temp = append(temp, node.Value)
			if node.Left != nil{
				st.PushBack(node.Left)
			}
			if node.Right != nil{
				st.PushBack(node.Right)
			}
		}
		res = append(res, temp)
		temp = []int{}  //每层的临时存储数组清空
	}
	return res
}
