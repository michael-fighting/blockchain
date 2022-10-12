package db

import "time"

const (
	TableVote       = "vote"
	TableVoteDetail = "vote_detail"
)

type CommonField struct {
	Id        int64      `gorm:"column:id;PRIMARY_KEY" json:"id"`
	CreatedAt time.Time  `gorm:"column:create_at" json:"createAt"`
	UpdatedAt time.Time  `gorm:"column:update_at" json:"updateAt"`
	DeleteAt  *time.Time `gorm:"column:delete_at;default:null" json:"deleteAt"`
}

type Vote struct {
	CommonField
	Title      string    // 投票项目标题
	State      string    // 投票状态
	PicUrl     string    //投票图片链接
	Start_time time.Time //投票开始时间
	End_time   time.Time //投票结束时间
	Rule       string    //投票规则
	Scenery    string    //投票场景
}

func (*Vote) TableName() string {
	return TableVote
}

type VoteDetail struct {
	CommonField
	VoteId      int64  //投票id
	PicItemUrl  string // 投票项图片链接
	Number      int    // 当前投票项投票数量
	Sum_Number  int    // 投票总数
	Description string //投票项说明
}

func (*VoteDetail) TableName() string {
	return TableVoteDetail
}
