---
title: 动手写ORM框架 - GeeORM第五天 实现钩子(Hooks)
date: 2020-03-08 18:00:00
description: 7天用 Go语言/golang 从零实现 ORM 框架 GeeORM 教程(7 days implement golang object relational mapping framework from scratch tutorial)，动手写 ORM 框架，参照 gorm, xorm 的实现。通过反射(reflect)获取结构体绑定的钩子(hooks)，并调用；支持增删查改(CRUD)前后调用钩子。
tags:
- Go
nav: 从零实现
categories:
- ORM框架 - GeeORM
keywords:
- Go语言
- 从零实现ORM框架
- database/sql
- sqlite
- hooks
- BeforeUpdate
image: post/geeorm/geeorm_sm.jpg
github: https://github.com/geektutu/7days-golang
book: 七天用Go从零实现系列
book_title: Day5 实现钩子
---

本文是[7天用Go从零实现ORM框架GeeORM](https://geektutu.com/post/geeorm.html)的第五篇。

- 通过反射(reflect)获取结构体绑定的钩子(hooks)，并调用。
- 支持增删查改(CRUD)前后调用钩子。**代码约50行**

## 1 Hook 机制

Hook，翻译为钩子，其主要思想是提前在可能增加功能的地方埋好(预设)一个钩子，当我们需要重新修改或者增加这个地方的逻辑的时候，把扩展的类或者方法挂载到这个点即可。钩子的应用非常广泛，例如 Github 支持的 travis 持续集成服务，当有 `git push` 事件发生时，会触发 travis 拉取新的代码进行构建。IDE 中钩子也非常常见，比如，当按下 `Ctrl + s` 后，自动格式化代码。再比如前端常用的 `hot reload` 机制，前端代码发生变更时，自动编译打包，通知浏览器自动刷新页面，实现所写即所得。

钩子机制设计的好坏，取决于扩展点选择的是否合适。例如对于持续集成来说，代码如果不发生变更，反复构建是没有意义的，因此钩子应设计在代码可能发生变更的地方，比如 MR、PR 合并前后。

那对于 ORM 框架来说，合适的扩展点在哪里呢？很显然，记录的增删查改前后都是非常合适的。

比如，我们设计一个 `Account` 类，`Account` 包含有一个隐私字段 `Password`，那么每次查询后都需要做脱敏处理，才能继续使用。如果提供了 `AfterQuery` 的钩子，查询后，自动地将 `Password` 字段的值脱敏，是不是能省去很多冗余的代码呢？

## 2 实现钩子

GeeORM 的钩子与结构体绑定，即每个结构体需要实现各自的钩子。hook 相关的代码实现在 `session/hooks.go` 中。

[day5-hooks/session/hooks.go](https://github.com/geektutu/7days-golang/tree/master/gee-orm/day5-hooks/session)

```go
package session

import (
	"geeorm/log"
	"reflect"
)

// Hooks constants
const (
	BeforeQuery  = "BeforeQuery"
	AfterQuery   = "AfterQuery"
	BeforeUpdate = "BeforeUpdate"
	AfterUpdate  = "AfterUpdate"
	BeforeDelete = "BeforeDelete"
	AfterDelete  = "AfterDelete"
	BeforeInsert = "BeforeInsert"
	AfterInsert  = "AfterInsert"
)

// CallMethod calls the registered hooks
func (s *Session) CallMethod(method string, value any) {
	fm := reflect.ValueOf(s.RefTable().Model).MethodByName(method)
	if value != nil {
		fm = reflect.ValueOf(value).MethodByName(method)
	}
	param := []reflect.Value{reflect.ValueOf(s)}
	if fm.IsValid() {
		if v := fm.Call(param); len(v) > 0 {
			if err, ok := v[0].Interface().(error); ok {
				log.Error(err)
			}
		}
	}
	return
}
```

- 钩子机制同样是通过反射来实现的，`s.RefTable().Model` 或 `value` 即当前会话正在操作的对象，使用 `MethodByName` 方法反射得到该对象的方法。
- 将 `s *Session` 作为入参调用。每一个钩子的入参类型均是 `*Session`。

接下来，将 `CallMethod()` 方法在 Find、Insert、Update、Delete 方法内部调用即可。例如，`Find` 方法修改为：

```go
// Find gets all eligible records
func (s *Session) Find(values any) error {
	s.CallMethod(BeforeQuery, nil)
    // ...
    for rows.Next() {
        dest := reflect.New(destType).Elem()
        // ...
        s.CallMethod(AfterQuery, dest.Addr().Interface())
        // ...
	}
	return rows.Close()
}
```

- `AfterQuery` 钩子可以操作每一行记录。

## 3 测试

新建 `session/hooks.go` 文件添加对应的测试用例。

```go
package session

import (
	"geeorm/log"
	"testing"
)

type Account struct {
	ID       int `geeorm:"PRIMARY KEY"`
	Password string
}

func (account *Account) BeforeInsert(s *Session) error {
	log.Info("before inert", account)
	account.ID += 1000
	return nil
}

func (account *Account) AfterQuery(s *Session) error {
	log.Info("after query", account)
	account.Password = "******"
	return nil
}

func TestSession_CallMethod(t *testing.T) {
	s := NewSession().Model(&Account{})
	_ = s.DropTable()
	_ = s.CreateTable()
	_, _ = s.Insert(&Account{1, "123456"}, &Account{2, "qwerty"})

	u := &Account{}

	err := s.First(u)
	if err != nil || u.ID != 1001 || u.Password != "******" {
		t.Fatal("Failed to call hooks after query, got", u)
	}
}
```

在这个测试用例中，测试了 `BeforeInsert` 和 `AfterQuery` 2 个钩子。

- `BeforeInsert` 将 account.ID 的值增加 1000
- `AfterQuery` 将密码脱敏，显示为 6 个 `*`。

## 附 推荐阅读

- [Go 语言简明教程](https://geektutu.com/post/quick-golang.html)
- [Go Test 单元测试简明教程](https://geektutu.com/post/quick-go-test.html)
- [SQLite 常用命令速查表](https://geektutu.com/post/cheat-sheet-sqlite.html)

## 白话复盘：Hook 是模型生命周期中的插槽

### 什么时候需要 Hook

有些逻辑与某个模型紧密相关，并且应在每次增删改查时自动发生，例如插入前补默认时间、查询后计算派生字段、删除前写审计记录。Hook 让模型通过约定名称的方法，在 ORM 操作前后获得执行机会，而不用修改 ORM 的核心代码。

教程定义了 `BeforeInsert`、`AfterInsert`、`BeforeQuery`、`AfterQuery` 等阶段。Session 在对应位置调用 `CallMethod`，后者用反射查看当前模型或具体记录是否实现同名方法；没有实现就直接跳过，因此 Hook 是可选扩展。

### 为什么有时对 Model 调用，有时对记录调用

查询前还没有具体结果，可以在 Session 当前模型上执行 `BeforeQuery`；每行数据扫描完成后，`AfterQuery` 更适合对刚得到的记录指针调用，这样 Hook 能修改该记录。理解“此刻有没有具体对象”就能理解 value 参数的选择。

指针接收者与值接收者的方法集合不同。需要在 Hook 中修改字段时通常应使用指针接收者，并确保反射拿到的是可寻址指针。

### 容易踩坑

- 教学实现会记录 Hook 返回的错误，但不会把它传回 Insert/Query 并中止操作；生产设计应明确错误是否阻止 SQL、是否触发回滚。
- Hook 中再次调用同类 ORM 操作可能递归触发自己，造成无限循环或重复副作用。
- 隐式逻辑越多，代码越难追踪。只把真正跨调用点且属于模型生命周期的规则放进 Hook。
- `AfterInsert` 执行时 SQL 已经发生；若它失败但不在事务里，前面的写入不会自动撤销。
- 反射按方法名约定工作，拼写或签名不匹配时不会自动得到编译期提示，应有测试覆盖。

### 动手检查

给模型添加 `BeforeInsert` 自动补字段，再在 `AfterQuery` 中记录一次调用。分别传值和指针，观察哪些方法能被找到；再让 Hook 返回错误，确认当前教学实现只是记录错误，由此思考你希望生产 API 怎样传播失败。
