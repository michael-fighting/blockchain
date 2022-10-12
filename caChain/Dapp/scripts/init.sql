
create TABLE vote (
                          `id` BIGINT(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键,投票id',
                          `title` VARCHAR(256) NOT NULL COMMENT '投票标题',
                          `state` VARCHAR(256) NOT NULL COMMENT '投票状态',
                          `picUrl` VARCHAR(256) NOT NULL COMMENT '投票图片链接',
                          `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                          `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
                          PRIMARY KEY (`id`))
)
ENGINE = InnoDB
DEFAULT CHARACTER SET = utf8mb4
COMMENT = '投票列表';

create TABLE voteDetail (
                          `id` BIGINT(10) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键,投票项id',
                          `voteId` BIGINT(10) UNSIGNED NOT NULL '投票id',
                          //`name` VARCHAR(256) NOT NULL COMMENT '投票项目名称',
                          `picItemUrl` VARCHAR(256) NOT NULL COMMENT '投票项图片链接',
                          `number` VARCHAR(10) NULL COMMENT '当前投票项投票数量',
                          `sum_Number` VARCHAR(10) NULL COMMENT '投票总数',
                          `description` VARCHAR(256) NOT NULL COMMENT '投票项说明',
                          `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                          `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
                          FOREIGN KEY(voteId) REFERENCES vote(id)
                          PRIMARY KEY (`id`))
)
ENGINE = InnoDB
DEFAULT CHARACTER SET = utf8mb4
COMMENT = '投票项详情列表';
