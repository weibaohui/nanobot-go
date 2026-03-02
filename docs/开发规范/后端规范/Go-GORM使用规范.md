# Go + GORM 使用规范

## 1. 模型定义规范

### 1.1 基础模型结构

```go
package models

import (
    "time"

    "gorm.io/gorm"
)

// BaseModel 所有模型的基础字段
type BaseModel struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
```

### 1.2 模型标签规范

```go
type User struct {
    BaseModel
    Username    string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username" binding:"required,min=3,max=50"`
    Email       string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email" binding:"required,email"`
    Password    string         `gorm:"type:varchar(255);not null" json:"-"` // json:"-" 防止密码泄露
    Status      string         `gorm:"type:varchar(20);default:'active';index" json:"status"`
    Profile     UserProfile    `gorm:"foreignKey:UserID" json:"profile,omitempty"`
    Orders      []Order        `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
    return "users"
}
```

### 1.3 标签使用说明

| 标签 | 用途 | 示例 |
|------|------|------|
| `primarykey` | 主键 | `gorm:"primarykey"` |
| `not null` | 非空约束 | `gorm:"not null"` |
| `uniqueIndex` | 唯一索引 | `gorm:"uniqueIndex"` |
| `index` | 普通索引 | `gorm:"index"` |
| `size:n` | 字段长度 | `gorm:"size:255"` |
| `type:type` | 数据库类型 | `gorm:"type:varchar(255)"` |
| `default:value` | 默认值 | `gorm:"default:'active'"` |
| `comment:text` | 字段注释 | `gorm:"comment:用户名"` |

## 2. 数据库操作规范

### 2.1 查询操作

```go
// 简单查询
func GetUserByID(db *gorm.DB, id uint) (*User, error) {
    var user User
    if err := db.First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// 条件查询 - 使用链式调用
func GetActiveUsers(db *gorm.DB, limit int) ([]User, error) {
    var users []User
    if err := db.
        Where("status = ?", "active").
        Order("created_at DESC").
        Limit(limit).
        Find(&users).Error; err != nil {
        return nil, err
    }
    return users, nil
}

// 使用结构体条件查询（零值会被忽略）
func GetUsersByFilter(db *gorm.DB, filter User) ([]User, error) {
    var users []User
    if err := db.Where(&filter).Find(&users).Error; err != nil {
        return nil, err
    }
    return users, nil
}

// 使用 Map 条件（包含零值）
func GetUsersWithZeroValue(db *gorm.DB) ([]User, error) {
    var users []User
    if err := db.Where(map[string]interface{}{
        "status": "", // 会查询 status = '' 的记录
    }).Find(&users).Error; err != nil {
        return nil, err
    }
    return users, nil
}

// 预加载关联数据
func GetUserWithOrders(db *gorm.DB, id uint) (*User, error) {
    var user User
    if err := db.
        Preload("Orders").
        Preload("Profile").
        First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// 预加载条件
func GetUserWithActiveOrders(db *gorm.DB, id uint) (*User, error) {
    var user User
    if err := db.
        Preload("Orders", "status = ?", "completed").
        First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// 分页查询
func ListUsers(db *gorm.DB, page, pageSize int) ([]User, int64, error) {
    var users []User
    var total int64

    offset := (page - 1) * pageSize

    // 获取总数
    if err := db.Model(&User{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 分页查询
    if err := db.
        Offset(offset).
        Limit(pageSize).
        Order("created_at DESC").
        Find(&users).Error; err != nil {
        return nil, 0, err
    }

    return users, total, nil
}
```

### 2.2 创建操作

```go
// 创建单条记录
func CreateUser(db *gorm.DB, user *User) error {
    return db.Create(user).Error
}

// 批量创建（高效）
func BatchCreateUsers(db *gorm.DB, users []User) error {
    if len(users) == 0 {
        return nil
    }
    return db.CreateInBatches(users, 100).Error // 每次插入100条
}

// 创建并返回ID
func CreateUserAndGetID(db *gorm.DB, user *User) (uint, error) {
    if err := db.Create(user).Error; err != nil {
        return 0, err
    }
    return user.ID, nil
}
```

### 2.3 更新操作

```go
// 更新单条记录 - 保存完整对象
func UpdateUser(db *gorm.DB, user *User) error {
    return db.Save(user).Error
}

// 更新指定字段 - 推荐方式
func UpdateUserEmail(db *gorm.DB, id uint, email string) error {
    return db.Model(&User{}).
        Where("id = ?", id).
        Update("email", email).Error
}

// 更新多个字段
func UpdateUserProfile(db *gorm.DB, id uint, updates map[string]interface{}) error {
    return db.Model(&User{}).
        Where("id = ?", id).
        Updates(updates).Error
}

// 使用结构体更新（零值会被忽略）
func UpdateUserWithStruct(db *gorm.DB, id uint, user User) error {
    return db.Model(&User{}).
        Where("id = ?", id).
        Omit("created_at", "deleted_at").
        Updates(user).Error
}

// 使用 Select 指定要更新的字段（包含零值）
func UpdateUserWithZeroValue(db *gorm.DB, id uint, status string) error {
    return db.Model(&User{}).
        Where("id = ?", id).
        Select("status").
        Updates(map[string]interface{}{"status": status}).Error
}

// 批量更新
func UpdateUsersStatus(db *gorm.DB, ids []uint, status string) error {
    return db.Model(&User{}).
        Where("id IN ?", ids).
        Update("status", status).Error
}
```

### 2.4 删除操作

```go
// 软删除（推荐）
func DeleteUser(db *gorm.DB, id uint) error {
    return db.Delete(&User{}, id).Error
}

// 永久删除
func HardDeleteUser(db *gorm.DB, id uint) error {
    return db.Unscoped().Delete(&User{}, id).Error
}

// 批量软删除
func DeleteUsers(db *gorm.DB, ids []uint) error {
    return db.Delete(&User{}, ids).Error
}

// 恢复软删除的数据
func RestoreUser(db *gorm.DB, id uint) error {
    return db.Unscoped().
        Model(&User{}).
        Where("id = ?", id).
        Update("deleted_at", nil).Error
}
```

## 3. 事务处理规范

### 3.1 基本事务

```go
func TransferMoney(db *gorm.DB, fromID, toID uint, amount float64) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 扣款
        if err := tx.Model(&Account{}).
            Where("id = ? AND balance >= ?", fromID, amount).
            Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
            return err
        }

        // 加款
        if err := tx.Model(&Account{}).
            Where("id = ?", toID).
            Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
            return err
        }

        return nil
    })
}
```

### 3.2 嵌套事务

```go
func CreateOrderWithItems(db *gorm.DB, order *Order, items []OrderItem) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 创建订单
        if err := tx.Create(order).Error; err != nil {
            return err
        }

        // 嵌套事务处理订单项
        if err := tx.Transaction(func(tx2 *gorm.DB) error {
            for _, item := range items {
                item.OrderID = order.ID
                if err := tx2.Create(&item).Error; err != nil {
                    return err
                }
            }
            return nil
        }); err != nil {
            return err
        }

        return nil
    })
}
```

### 3.3 事务回滚和提交

```go
func ManualTransaction(db *gorm.DB) error {
    tx := db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r)
        }
    }()

    if err := tx.Error; err != nil {
        return err
    }

    // 执行操作
    if err := tx.Create(&User{}).Error; err != nil {
        tx.Rollback()
        return err
    }

    // 提交事务
    return tx.Commit().Error
}
```

## 4. 钩子函数规范

```go
type User struct {
    BaseModel
    Username string `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
    Password string `gorm:"type:varchar(255);not null" json:"-"`
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
    // 密码加密
    if u.Password != "" {
        u.Password = hashPassword(u.Password)
    }
    return nil
}

// BeforeUpdate 更新前钩子
func (u *User) BeforeUpdate(tx *gorm.DB) error {
    // 检查是否修改了密码，如果是则加密
    if tx.Statement.Changed("Password") && u.Password != "" {
        u.Password = hashPassword(u.Password)
    }
    return nil
}

// AfterCreate 创建后钩子
func (u *User) AfterCreate(tx *gorm.DB) error {
    // 发送欢迎邮件
    go sendWelcomeEmail(u.Email)
    return nil
}

// AfterDelete 删除后钩子
func (u *User) AfterDelete(tx *gorm.DB) error {
    // 清理关联数据
    return tx.Where("user_id = ?", u.ID).Delete(&Profile{}).Error
}
```

## 5. 错误处理规范

```go
import "errors"

var (
    ErrRecordNotFound = errors.New("record not found")
)

// 标准错误处理
func GetUser(db *gorm.DB, id uint) (*User, error) {
    var user User
    if err := db.First(&user, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrRecordNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return &user, nil
}

// 错误检查和包装
func CreateUserWithCheck(db *gorm.DB, user *User) error {
    if err := db.Create(user).Error; err != nil {
        if errors.Is(err, gorm.ErrDuplicatedKey) {
            return fmt.Errorf("user already exists: %w", err)
        }
        return fmt.Errorf("failed to create user: %w", err)
    }
    return nil
}
```

## 6. 性能优化建议

### 6.1 索引优化

```go
// 复合索引
type Order struct {
    BaseModel
    UserID    uint      `gorm:"index:idx_user_status" json:"user_id"`
    Status    string    `gorm:"size:20;index:idx_user_status;index" json:"status"`
    CreatedAt time.Time `gorm:"index" json:"created_at"`
}
```

### 6.2 查询优化

```go
// 只选择需要的字段
func GetUserBasicInfo(db *gorm.DB, id uint) (*User, error) {
    var user User
    if err := db.
        Select("id", "username", "status").
        First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// 使用 Pluck 获取单列数据
func GetAllUserIDs(db *gorm.DB) ([]uint, error) {
    var ids []uint
    if err := db.Model(&User{}).Pluck("id", &ids).Error; err != nil {
        return nil, err
    }
    return ids, nil
}

// 批量查询避免 N+1 问题
func GetUsersWithOrders(db *gorm.DB, userIDs []uint) (map[uint][]Order, error) {
    var orders []Order
    if err := db.
        Where("user_id IN ?", userIDs).
        Find(&orders).Error; err != nil {
        return nil, err
    }

    result := make(map[uint][]Order)
    for _, order := range orders {
        result[order.UserID] = append(result[order.UserID], order)
    }
    return result, nil
}
```

## 7. 数据库迁移规范

```go
// 自动迁移
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &User{},
        &Profile{},
        &Order{},
        &OrderItem{},
    )
}

// 建议在开发环境使用 AutoMigrate
// 生产环境使用迁移脚本（如 golang-migrate）
```

## 8. 命名约定

| 类型 | 约定 | 示例 |
|------|------|------|
| 结构体 | PascalCase | `UserProfile` |
| 方法 | PascalCase | `GetUserByID` |
| 变量 | camelCase | `userName` |
| 常量 | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT` |
| 表名 | snake_case | `user_profiles` |
| 字段名 | snake_case | `created_at` |

## 9. 安全规范

### 9.1 防止 SQL 注入

```go
// ✅ 正确：使用参数化查询
db.Where("username = ?", username).First(&user)

// ❌ 错误：直接拼接字符串
db.Where(fmt.Sprintf("username = '%s'", username)).First(&user)
```

### 9.2 敏感字段处理

```go
type User struct {
    BaseModel
    Username string `gorm:"type:varchar(50)" json:"username"`
    Password string `gorm:"type:varchar(255)" json:"-"` // json:"-" 防止序列化
    Token    string `gorm:"type:varchar(255)" json:"-" binding:"-"` // binding:"-" 不参与参数绑定
}
```

## 10. 测试规范

### 10.1 使用测试数据库

```go
func TestCreateUser(t *testing.T) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    assert.NoError(t, err)

    // 自动迁移
    err = db.AutoMigrate(&User{})
    assert.NoError(t, err)

    // 测试用例
    user := &User{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "hashedpassword",
    }
    err = db.Create(user).Error
    assert.NoError(t, err)
    assert.NotZero(t, user.ID)
}
```

### 10.2 使用事务回滚

```go
func TestUserService(t *testing.T) {
    db, err := setupTestDB()
    assert.NoError(t, err)

    // 每个测试都在事务中执行，测试完成后回滚
    db.Transaction(func(tx *gorm.DB) error {
        // 测试代码
        user := &User{Username: "test"}
        err := tx.Create(user).Error
        assert.NoError(t, err)

        // 查询验证
        var found User
        err = tx.First(&found, user.ID).Error
        assert.NoError(t, err)
        assert.Equal(t, "test", found.Username)

        // 返回错误会自动回滚
        return nil
    })
}
```

## 11. 最佳实践总结

1. **始终使用 `*gorm.DB` 而不是 `gorm.DB`** - `gorm.DB` 是线程安全的，复制后每个副本有独立的状态
2. **使用链式调用** - GORM 的链式调用让代码更清晰
3. **错误处理** - 始终检查 `Error`，不要假设操作成功
4. **事务处理** - 对于多步操作，使用事务保证数据一致性
5. **索引优化** - 为常用查询字段添加索引
6. **避免 N+1 问题** - 使用 `Preload` 或批量查询
7. **字段保护** - 使用 `json:"-"` 保护敏感字段
8. **钩子谨慎使用** - 钩子会增加隐式行为，确保有文档说明
9. **迁移策略** - 开发用 AutoMigrate，生产用迁移脚本
10. **测试隔离** - 每个测试使用独立的事务，测试后回滚