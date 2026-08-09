---
title: 动手写ORM框架 - GeeORM第四天 链式操作与更新删除
date: 2020-03-08 16:00:00
description: 7天用 Go语言/golang 从零实现 ORM 框架 GeeORM 教程(7 days implement golang object relational mapping framework from scratch tutorial)，动手写 ORM 框架，参照 gorm, xorm 的实现。通过链式(chain)操作，支持查询条件(where, order by, limit 等)的叠加；实现记录的更新(update)、删除(delete)和统计(count)功能。
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
- chain operation
- delete from
image: post/geeorm/geeorm_sm.jpg
github: https://github.com/geektutu/7days-golang
book: 七天用Go从零实现系列
book_title: Day4 链式操作与更新删除
---

本文是[7天用Go从零实现ORM框架GeeORM](https://geektutu.com/post/geeorm.html)的第四篇。

- 通过链式(chain)操作，支持查询条件(where, order by, limit 等)的叠加。
- 实现记录的更新(update)、删除(delete)和统计(count)功能。**代码约100行**

## 1 支持 Update、Delete 和 Count

### 1.1 子句生成器

clause 负责构造 SQL 语句，如果需要增加对更新(update)、删除(delete)和统计(count)功能的支持，第一步自然是在 clause 中实现 update、delete 和 count 子句的生成器。

第一步：在原来的基础上，新增 UPDATE、DELETE、COUNT 三个 `Type` 类型的枚举值。

[day4-chain-operation/clause/clause.go](https://github.com/geektutu/7days-golang/tree/master/gee-orm/day4-chain-operation/clause)

```go
// Support types for Clause
const (
	INSERT Type = iota
	VALUES
	SELECT
	LIMIT
	WHERE
	ORDERBY
	UPDATE
	DELETE
	COUNT
)
```

第二步：实现对应字句的 generator，并注册到全局变量 `generators` 中

[day4-chain-operation/clause/generator.go](https://github.com/geektutu/7days-golang/tree/master/gee-orm/day4-chain-operation/clause)

```go
func init() {
	generators = make(map[Type]generator)
	generators[INSERT] = _insert
	generators[VALUES] = _values
	generators[SELECT] = _select
	generators[LIMIT] = _limit
	generators[WHERE] = _where
	generators[ORDERBY] = _orderBy
	generators[UPDATE] = _update
	generators[DELETE] = _delete
	generators[COUNT] = _count
}

func _update(values ...any) (string, []any) {
	tableName := values[0]
	m := values[1].(map[string]any)
	var keys []string
	var vars []any
	for k, v := range m {
		keys = append(keys, k+" = ?")
		vars = append(vars, v)
	}
	return fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(keys, ", ")), vars
}

func _delete(values ...any) (string, []any) {
	return fmt.Sprintf("DELETE FROM %s", values[0]), []any{}
}

func _count(values ...any) (string, []any) {
	return _select(values[0], []string{"count(*)"})
}
```

- `_update` 设计入参是2个，第一个参数是表名(table)，第二个参数是 map 类型，表示待更新的键值对。
- `_delete` 只有一个入参，即表名。
- `_count` 只有一个入参，即表名，并复用了 `_select` 生成器。


### 1.2 Update 方法

子句的 generator 已经准备好了，接下来和 Insert、Find 等方法一样，在 `session/record.go` 中按照一定顺序拼接 SQL 语句并调用就可以了。

[day4-chain-operation/session/record.go](https://github.com/geektutu/7days-golang/tree/master/gee-orm/day4-chain-operation/session)

```go
// support map[string]any
// also support kv list: "Name", "Tom", "Age", 18, ....
func (s *Session) Update(kv ...any) (int64, error) {
	m, ok := kv[0].(map[string]any)
	if !ok {
		m = make(map[string]any)
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
	}
	s.clause.Set(clause.UPDATE, s.RefTable().Name, m)
	sql, vars := s.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
```

Update 方法比较特别的一点在于，Update 接受 2 种入参，平铺开来的键值对和 map 类型的键值对。因为 generator 接受的参数是 map 类型的键值对，因此 `Update` 方法会动态地判断传入参数的类型，如果是不是 map 类型，则会自动转换。


### 1.3 Delete 方法

```go
// Delete records with where clause
func (s *Session) Delete() (int64, error) {
	s.clause.Set(clause.DELETE, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.DELETE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
```

### 1.4 Count 方法

```go
// Count records with where clause
func (s *Session) Count() (int64, error) {
	s.clause.Set(clause.COUNT, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.COUNT, clause.WHERE)
	row := s.Raw(sql, vars...).QueryRow()
	var tmp int64
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}
	return tmp, nil
}
```

## 2 链式调用(chain)

链式调用是一种简化代码的编程方式，能够使代码更简洁、易读。链式调用的原理也非常简单，某个对象调用某个方法后，将该对象的引用/指针返回，即可以继续调用该对象的其他方法。通常来说，当某个对象需要一次调用多个方法来设置其属性时，就非常适合改造为链式调用了。

SQL 语句的构造过程就非常符合这个条件。SQL 语句由多个子句构成，典型的例如 SELECT 语句，往往需要设置查询条件（WHERE）、限制返回行数（LIMIT）等。理想的调用方式应该是这样的：

```go
s := geeorm.NewEngine("sqlite", "gee.db").NewSession()
var users []User
s.Where("Age > 18").Limit(3).Find(&users)
```

从上面的示例中，可以看出，`WHERE`、`LIMIT`、`ORDER BY` 等查询条件语句非常适合链式调用。这几个子句的 generator 在之前就已经实现了，那我们接下来在 `session/record.go` 中添加对应的方法即可。

[day4-chain-operation/session/record.go](https://github.com/geektutu/7days-golang/tree/master/gee-orm/day4-chain-operation/session)

```go
// Limit adds limit condition to clause
func (s *Session) Limit(num int) *Session {
	s.clause.Set(clause.LIMIT, num)
	return s
}

// Where adds limit condition to clause
func (s *Session) Where(desc string, args ...any) *Session {
	var vars []any
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}

// OrderBy adds order by condition to clause
func (s *Session) OrderBy(desc string) *Session {
	s.clause.Set(clause.ORDERBY, desc)
	return s
}
```

## 3 First 只返回一条记录

很多时候，我们期望 SQL 语句只返回一条记录，比如根据某个童鞋的学号查询他的信息，返回结果有且只有一条。结合链式调用，我们可以非常容易地实现 First 方法。

```go
func (s *Session) First(value any) error {
	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()
	if err := s.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}
	if destSlice.Len() == 0 {
		return errors.New("NOT FOUND")
	}
	dest.Set(destSlice.Index(0))
	return nil
}
```

First 方法可以这么使用：

```go
u := &User{}
_ = s.OrderBy("Age DESC").First(u)
```

> 实现原理：根据传入的类型，利用反射构造切片，调用 `Limit(1)` 限制返回的行数，调用 `Find` 方法获取到查询结果。

## 4 测试

接下来呢，我们在 `record_test.go` 中添加几个测试用例，检测功能是否正常。

```go
package session

import "testing"

var (
	user1 = &User{"Tom", 18}
	user2 = &User{"Sam", 25}
	user3 = &User{"Jack", 25}
)

func testRecordInit(t *testing.T) *Session {
	t.Helper()
	s := NewSession().Model(&User{})
	err1 := s.DropTable()
	err2 := s.CreateTable()
	_, err3 := s.Insert(user1, user2)
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("failed init test records")
	}
	return s
}

func TestSession_Limit(t *testing.T) {
	s := testRecordInit(t)
	var users []User
	err := s.Limit(1).Find(&users)
	if err != nil || len(users) != 1 {
		t.Fatal("failed to query with limit condition")
	}
}

func TestSession_Update(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Update("Age", 30)
	u := &User{}
	_ = s.OrderBy("Age DESC").First(u)

	if affected != 1 || u.Age != 30 {
		t.Fatal("failed to update")
	}
}

func TestSession_DeleteAndCount(t *testing.T) {
	s := testRecordInit(t)
	affected, _ := s.Where("Name = ?", "Tom").Delete()
	count, _ := s.Count()

	if affected != 1 || count != 1 {
		t.Fatal("failed to delete or count")
	}
}
```

## 附 推荐阅读

- [Go 语言简明教程](https://geektutu.com/post/quick-golang.html)
- [Go Test 单元测试简明教程](https://geektutu.com/post/quick-go-test.html)
- [SQLite 常用命令速查表](https://geektutu.com/post/cheat-sheet-sqlite.html)

## 白话复盘：链式调用是在逐步填写一张 SQL 表单

### Session 为什么能一路点下去

`Where`、`Limit`、`OrderBy` 并没有立刻访问数据库，它们只是把对应 Clause 写进当前 Session，并返回同一个 Session 指针。直到调用 `Find`、`Update`、`Delete` 或 `Count`，框架才按 SQL 语法规定的顺序把片段拼起来并执行。

因此 `s.Where(...).OrderBy(...).Limit(1).Find(&users)` 可以理解为先填条件、排序、数量三栏，最后按下“执行”按钮。方法在代码里的调用顺序不一定就是 SQL 片段的输出顺序，最终顺序由生成器明确控制。

### 为什么执行后要 `Clear`

Session 保存的是可变构建状态。如果上一条语句的 WHERE 残留到下一条 DELETE，就可能误删或漏删数据。执行结束后清空 SQL、变量与 Clause，能让同一个 Session 开始下一次构建时回到干净状态。

不过这也说明 Session 不适合被多个 goroutine 同时链式使用。即使底层 `*sql.DB` 可以并发共享，同一个可变 Session 仍会互相覆盖条件和参数；并发任务应分别创建 Session。

### Update 的两种输入

教程允许传 map，或按 `key, value` 成对传参。无论入口形式如何，最终都要生成 `SET field=?, ...` 与同序参数。Go 的 map 遍历顺序不固定不是问题，只要列名和变量在同一次遍历中成对追加；但测试不要死板依赖 map 生成字段的固定排列。

### 容易踩坑

- 没有 WHERE 的 Update/Delete 可能影响整张表。生产 ORM 常提供显式保护或要求调用者确认全表操作。
- `First` 往往通过 `LIMIT 1` 实现，但没有 ORDER BY 时“第一条”并没有稳定业务含义。
- 链式 API 看起来像不可变值，实际却在修改 Session；保留中间变量并交叉复用会产生意外状态。
- 参数仍必须使用占位符，`OrderBy` 等结构片段若接收用户输入则要使用白名单，参数化不能保护列名或关键字。

### 动手检查

连续用同一 Session 执行两条条件完全不同的查询，并打印最终 SQL，确认第二条没有继承第一条 WHERE。再尝试无 WHERE 的 Delete，在测试数据库中观察风险，理解生产保护为何必要。
