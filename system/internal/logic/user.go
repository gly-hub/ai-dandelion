package logic

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type UserLogic struct {
	userDao           *dao.User
	roleDao           *dao.Role
	tokenStore        *dao.AuthTokenStore
	operationLogLogic *OperationLogLogic
	accessTokenSecret string
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
}

type AuthResult struct {
	User             *systemproto.User
	Roles            []*systemproto.Role
	AccessToken      string
	RefreshToken     string
	AccessExpiresIn  int64
	RefreshExpiresIn int64
}

func NewUserLogic(userDao *dao.User, roleDao *dao.Role, accessTokenSecret string, accessTokenTTL time.Duration, operationLogs ...*OperationLogLogic) *UserLogic {
	return newUserLogic(userDao, roleDao, nil, accessTokenSecret, accessTokenTTL, 7*24*time.Hour, operationLogs...)
}

func NewUserLogicWithTokenStore(userDao *dao.User, roleDao *dao.Role, tokenStore *dao.AuthTokenStore, accessTokenSecret string, accessTokenTTL, refreshTokenTTL time.Duration, operationLogs ...*OperationLogLogic) *UserLogic {
	return newUserLogic(userDao, roleDao, tokenStore, accessTokenSecret, accessTokenTTL, refreshTokenTTL, operationLogs...)
}

func newUserLogic(userDao *dao.User, roleDao *dao.Role, tokenStore *dao.AuthTokenStore, accessTokenSecret string, accessTokenTTL, refreshTokenTTL time.Duration, operationLogs ...*OperationLogLogic) *UserLogic {
	logic := &UserLogic{
		userDao:           userDao,
		roleDao:           roleDao,
		tokenStore:        tokenStore,
		accessTokenSecret: accessTokenSecret,
		accessTokenTTL:    accessTokenTTL,
		refreshTokenTTL:   refreshTokenTTL,
	}
	if len(operationLogs) > 0 {
		logic.operationLogLogic = operationLogs[0]
	}
	return logic
}

func (u *UserLogic) Login(ctx context.Context, req *systemproto.LoginReq) (*AuthResult, error) {
	username, err := validateUsername(req.GetUsername())
	if err != nil {
		return nil, err
	}
	password, err := validatePassword(req.GetPassword(), true)
	if err != nil {
		return nil, err
	}
	user, err := u.userDao.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}
	if user.Status != model.UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}
	roles, err := u.listUserRolesProto(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !hasEnabledRoles(roles) {
		return nil, errors.New("用户未分配角色，请联系管理员")
	}
	return u.issueNewSessionTokens(ctx, user, roles)
}

func (u *UserLogic) RefreshToken(ctx context.Context, req *systemproto.RefreshTokenReq) (*AuthResult, error) {
	if u.tokenStore == nil {
		return nil, errors.New("auth token store is unavailable")
	}
	refreshToken := strings.TrimSpace(req.GetRefreshToken())
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}
	sessionID, err := u.tokenStore.ConsumeRefreshSessionID(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	session, err := u.tokenStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	user, err := u.userDao.Get(ctx, session.UserID)
	if err != nil {
		return nil, wrapNotFoundError(err)
	}
	if user.Status != model.UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}
	roles, err := u.listUserRolesProto(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !hasEnabledRoles(roles) {
		return nil, errors.New("用户未分配角色，请联系管理员")
	}
	remaining := time.Until(time.Unix(session.ExpiresAt, 0))
	if remaining <= 0 {
		return nil, errors.New("invalid or expired refresh token")
	}
	return u.issueRotatedTokens(ctx, user, roles, sessionID, remaining)
}

func (u *UserLogic) Logout(ctx context.Context, req *systemproto.LogoutReq) error {
	if u.tokenStore == nil {
		return errors.New("auth token store is unavailable")
	}
	refreshToken := strings.TrimSpace(req.GetRefreshToken())
	if refreshToken == "" {
		return nil
	}
	sessionID, err := u.tokenStore.ConsumeRefreshSessionID(ctx, refreshToken)
	if err != nil {
		return nil
	}
	return u.tokenStore.RevokeSession(ctx, sessionID)
}

func (u *UserLogic) ValidateToken(ctx context.Context, req *systemproto.ValidateTokenReq) (*systemproto.User, []*systemproto.Role, error) {
	if u.tokenStore == nil {
		return nil, nil, errors.New("auth token store is unavailable")
	}
	claims, err := authctx.VerifyAccessToken(u.accessTokenSecret, req.GetToken())
	if err != nil {
		return nil, nil, err
	}
	sessionID, err := u.tokenStore.AccessSessionID(ctx, claims.ID)
	if err != nil {
		return nil, nil, authctx.ErrInvalidToken
	}
	if claims.SessionID == "" || claims.SessionID != sessionID {
		return nil, nil, authctx.ErrInvalidToken
	}
	session, err := u.tokenStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, authctx.ErrExpiredToken
	}
	user, err := u.userDao.Get(ctx, session.UserID)
	if err != nil {
		return nil, nil, wrapNotFoundError(err)
	}
	if user.Status != model.UserStatusEnabled {
		return nil, nil, errors.New("user is disabled")
	}
	roles, err := u.listUserRolesProto(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if !hasEnabledRoles(roles) {
		return nil, nil, errors.New("用户未分配角色，请联系管理员")
	}
	return modelUserToProto(user, roleIDsFromProto(roles)), roles, nil
}

func (u *UserLogic) issueNewSessionTokens(ctx context.Context, user *model.User, roles []*systemproto.Role) (*AuthResult, error) {
	if u.tokenStore == nil {
		return nil, errors.New("auth token store is unavailable")
	}
	refreshTTL := u.refreshTokenTTL
	if refreshTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}
	sessionID := uuid.NewString()
	accessTTL := minDuration(u.accessTokenTTL, refreshTTL)
	accessToken, accessJTI, err := authctx.SignAccessToken(u.accessTokenSecret, authctx.User{ID: user.ID, Username: user.Username}, sessionID, accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := authctx.NewOpaqueToken(32)
	if err != nil {
		return nil, err
	}
	if err := u.tokenStore.CreateSession(ctx, dao.AuthTokenSession{ID: sessionID, UserID: user.ID, Username: user.Username, ExpiresAt: time.Now().Add(refreshTTL).Unix()}, accessJTI, refreshToken, accessTTL, refreshTTL); err != nil {
		return nil, err
	}
	return &AuthResult{
		User:             modelUserToProto(user, roleIDsFromProto(roles)),
		Roles:            roles,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  authctx.TokenTTLSeconds(accessTTL),
		RefreshExpiresIn: authctx.TokenTTLSeconds(refreshTTL),
	}, nil
}

func (u *UserLogic) issueRotatedTokens(ctx context.Context, user *model.User, roles []*systemproto.Role, sessionID string, refreshTTL time.Duration) (*AuthResult, error) {
	if u.tokenStore == nil {
		return nil, errors.New("auth token store is unavailable")
	}
	if refreshTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}
	accessTTL := minDuration(u.accessTokenTTL, refreshTTL)
	accessToken, accessJTI, err := authctx.SignAccessToken(u.accessTokenSecret, authctx.User{ID: user.ID, Username: user.Username}, sessionID, accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := authctx.NewOpaqueToken(32)
	if err != nil {
		return nil, err
	}
	if err := u.tokenStore.SaveRotatedTokens(ctx, sessionID, accessJTI, refreshToken, accessTTL, refreshTTL); err != nil {
		return nil, err
	}
	return &AuthResult{
		User:             modelUserToProto(user, roleIDsFromProto(roles)),
		Roles:            roles,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  authctx.TokenTTLSeconds(accessTTL),
		RefreshExpiresIn: authctx.TokenTTLSeconds(refreshTTL),
	}, nil
}

func minDuration(left, right time.Duration) time.Duration {
	if left <= 0 {
		return right
	}
	if right <= 0 || left < right {
		return left
	}
	return right
}

func (u *UserLogic) ListUsers(ctx context.Context, _ *systemproto.ListUsersReq) ([]*systemproto.User, error) {
	users, err := u.userDao.List(ctx)
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
	}
	roleMap, err := u.roleDao.ListRoleIDsByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*systemproto.User, 0, len(users))
	for i := range users {
		out = append(out, modelUserToProto(&users[i], roleMap[users[i].ID]))
	}
	return out, nil
}

func (u *UserLogic) CreateUser(ctx context.Context, req *systemproto.CreateUserReq) (*systemproto.User, error) {
	username, err := validateUsername(req.GetUsername())
	if err != nil {
		return nil, err
	}
	email, err := validateEmail(req.GetEmail())
	if err != nil {
		return nil, err
	}
	password, err := validatePassword(req.GetPassword(), true)
	if err != nil {
		return nil, err
	}
	status, err := normalizeStatus(req.GetStatus(), model.UserStatusEnabled)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := nowUnixMicro()
	user := &model.User{
		ID:           uuid.NewString(),
		Username:     username,
		Email:        email,
		Phone:        strings.TrimSpace(req.GetPhone()),
		PasswordHash: string(hash),
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	afterData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}
	if err := u.userDao.Transaction(ctx, func(userDao *dao.User) error {
		if err := userDao.Create(ctx, user); err != nil {
			return err
		}
		return u.recordUserChange(ctx, dao.NewOperationLog(userDao.DB()), "user.create", "创建成员", user, "", afterData)
	}); err != nil {
		return nil, wrapDuplicateError(err)
	}
	return modelUserToProto(user, nil), nil
}

func (u *UserLogic) UpdateUser(ctx context.Context, req *systemproto.UpdateUserReq) (*systemproto.User, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}

	user, err := u.userDao.Get(ctx, id)
	if err != nil {
		return nil, wrapNotFoundError(err)
	}
	beforeData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}

	username, err := validateUsername(req.GetUsername())
	if err != nil {
		return nil, err
	}
	email, err := validateEmail(req.GetEmail())
	if err != nil {
		return nil, err
	}
	status, err := normalizeStatus(req.GetStatus(), user.Status)
	if err != nil {
		return nil, err
	}

	user.Username = username
	user.Email = email
	user.Phone = strings.TrimSpace(req.GetPhone())
	user.Status = status
	user.UpdatedAt = nowUnixMicro()

	if password := strings.TrimSpace(req.GetPassword()); password != "" {
		validated, err := validatePassword(password, true)
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(validated), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	afterData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}
	if err := u.userDao.Transaction(ctx, func(userDao *dao.User) error {
		if err := userDao.Update(ctx, user); err != nil {
			return err
		}
		return u.recordUserChange(ctx, dao.NewOperationLog(userDao.DB()), "user.update", "更新成员", user, beforeData, afterData)
	}); err != nil {
		return nil, wrapDuplicateError(err)
	}
	return u.modelUserWithRoles(ctx, user)
}

func (u *UserLogic) DeleteUser(ctx context.Context, req *systemproto.DeleteUserReq) error {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("id is required")
	}
	user, err := u.userDao.Get(ctx, id)
	if err != nil {
		return wrapNotFoundError(err)
	}
	beforeData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return err
	}
	if err := u.userDao.Transaction(ctx, func(userDao *dao.User) error {
		if err := userDao.Delete(ctx, id); err != nil {
			return err
		}
		return u.recordUserChange(ctx, dao.NewOperationLog(userDao.DB()), "user.delete", "删除成员", user, beforeData, "")
	}); err != nil {
		return wrapNotFoundError(err)
	}
	return nil
}

func (u *UserLogic) EnableUser(ctx context.Context, req *systemproto.EnableUserReq) (*systemproto.User, error) {
	return u.setUserStatus(ctx, req.GetId(), model.UserStatusEnabled)
}

func (u *UserLogic) DisableUser(ctx context.Context, req *systemproto.DisableUserReq) (*systemproto.User, error) {
	return u.setUserStatus(ctx, req.GetId(), model.UserStatusDisabled)
}

func (u *UserLogic) setUserStatus(ctx context.Context, id string, status int) (*systemproto.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	user, err := u.userDao.Get(ctx, id)
	if err != nil {
		return nil, wrapNotFoundError(err)
	}
	beforeData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}
	user.Status = status
	user.UpdatedAt = nowUnixMicro()
	afterData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}
	action, actionLabel := "user.enable", "启用成员"
	if status == model.UserStatusDisabled {
		action, actionLabel = "user.disable", "禁用成员"
	}
	if err := u.userDao.Transaction(ctx, func(userDao *dao.User) error {
		if err := userDao.UpdateStatus(ctx, id, status, user.UpdatedAt); err != nil {
			return err
		}
		return u.recordUserChange(ctx, dao.NewOperationLog(userDao.DB()), action, actionLabel, user, beforeData, afterData)
	}); err != nil {
		return nil, wrapNotFoundError(err)
	}
	return u.modelUserWithRoles(ctx, user)
}

func (u *UserLogic) GetUserRoles(ctx context.Context, req *systemproto.GetUserRolesReq) ([]string, []*systemproto.Role, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, nil, errors.New("userId is required")
	}
	if _, err := u.userDao.Get(ctx, userID); err != nil {
		return nil, nil, wrapNotFoundError(err)
	}
	roleIDs, err := u.roleDao.ListRoleIDsByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	roles, err := u.listUserRolesProto(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return roleIDs, roles, nil
}

func (u *UserLogic) SetUserRoles(ctx context.Context, req *systemproto.SetUserRolesReq) ([]string, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, errors.New("userId is required")
	}
	user, err := u.userDao.Get(ctx, userID)
	if err != nil {
		return nil, wrapNotFoundError(err)
	}
	beforeData, err := u.userAuditJSON(ctx, u.roleDao, user)
	if err != nil {
		return nil, err
	}
	roleIDs, err := u.normalizeRoleIDs(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}
	if err := u.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.SetUserRoles(ctx, userID, roleIDs, nowUnixMicro()); err != nil {
			return err
		}
		afterData, err := u.userAuditJSON(ctx, roleDao, user)
		if err != nil {
			return err
		}
		return u.recordUserChange(ctx, dao.NewOperationLog(roleDao.DB()), "user.role.update", "更新成员角色", user, beforeData, afterData)
	}); err != nil {
		return nil, err
	}
	return roleIDs, nil
}

type userAuditSnapshot struct {
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Phone    string          `json:"phone"`
	Status   int             `json:"status"`
	Roles    []userAuditRole `json:"roles"`
}

type userAuditRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

func (u *UserLogic) userAuditJSON(ctx context.Context, roleDao *dao.Role, user *model.User) (string, error) {
	roles, err := roleDao.ListRolesByUser(ctx, user.ID)
	if err != nil {
		return "", err
	}
	snapshot := userAuditSnapshot{Username: user.Username, Email: user.Email, Phone: user.Phone, Status: user.Status, Roles: make([]userAuditRole, 0, len(roles))}
	for _, role := range roles {
		snapshot.Roles = append(snapshot.Roles, userAuditRole{ID: role.ID, Name: role.Name, Code: role.Code})
	}
	data, err := json.Marshal(snapshot)
	return string(data), err
}

func (u *UserLogic) recordUserChange(ctx context.Context, operationLogDao *dao.OperationLog, action, actionLabel string, user *model.User, beforeData, afterData string) error {
	if u.operationLogLogic == nil {
		return nil
	}
	operatorName := "系统"
	if operator, ok := authctx.CurrentUser(ctx); ok && strings.TrimSpace(operator.Username) != "" {
		operatorName = operator.Username
	}
	return u.operationLogLogic.RecordWithDAO(ctx, operationLogDao, OperationLogInput{
		Module: OperationModuleSystem, Action: action, ActionLabel: actionLabel,
		ResourceType: OperationResourceUser, ResourceID: user.ID, ResourceName: user.Username,
		Summary: operatorName + actionLabel + "「" + user.Username + "」", BeforeData: beforeData, AfterData: afterData,
	})
}

func hasEnabledRoles(roles []*systemproto.Role) bool {
	for _, role := range roles {
		if role.GetStatus() == int32(model.RoleStatusEnabled) {
			return true
		}
	}
	return false
}

func roleIDsFromProto(roles []*systemproto.Role) []string {
	roleIDs := make([]string, 0, len(roles))
	for _, role := range roles {
		if role.GetId() == "" || role.GetStatus() != int32(model.RoleStatusEnabled) {
			continue
		}
		roleIDs = append(roleIDs, role.GetId())
	}
	return roleIDs
}

func (u *UserLogic) listUserRolesProto(ctx context.Context, userID string) ([]*systemproto.Role, error) {
	roles, err := u.roleDao.ListRolesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*systemproto.Role, 0, len(roles))
	for i := range roles {
		out = append(out, &systemproto.Role{
			Id:        roles[i].ID,
			Name:      roles[i].Name,
			Code:      roles[i].Code,
			Status:    int32(roles[i].Status),
			Remark:    roles[i].Remark,
			Sort:      int32(roles[i].Sort),
			CreatedAt: roles[i].CreatedAt,
		})
	}
	return out, nil
}

func (u *UserLogic) normalizeRoleIDs(ctx context.Context, roleIDs []string) ([]string, error) {
	unique := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		if _, err := u.roleDao.Get(ctx, roleID); err != nil {
			return nil, errors.New("role not found: " + roleID)
		}
		seen[roleID] = struct{}{}
		unique = append(unique, roleID)
	}
	return unique, nil
}

func (u *UserLogic) modelUserWithRoles(ctx context.Context, user *model.User) (*systemproto.User, error) {
	roleIDs, err := u.roleDao.ListRoleIDsByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return modelUserToProto(user, roleIDs), nil
}

func validateUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", errors.New("username is required")
	}
	if len(username) < 3 || len(username) > 64 {
		return "", errors.New("username length must be between 3 and 64")
	}
	return username, nil
}

func validateEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", errors.New("email is required")
	}
	if !emailPattern.MatchString(email) {
		return "", errors.New("email format is invalid")
	}
	return email, nil
}

func validatePassword(password string, required bool) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		if required {
			return "", errors.New("password is required")
		}
		return "", nil
	}
	if len(password) < 6 {
		return "", errors.New("password length must be at least 6")
	}
	return password, nil
}

func normalizeStatus(status int32, fallback int) (int, error) {
	if status == 0 {
		return fallback, nil
	}
	switch int(status) {
	case model.UserStatusEnabled, model.UserStatusDisabled:
		return int(status), nil
	default:
		return 0, errors.New("status must be 1 (enabled) or 2 (disabled)")
	}
}

func modelUserToProto(user *model.User, roleIDs []string) *systemproto.User {
	if user == nil {
		return nil
	}
	return &systemproto.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Phone:     user.Phone,
		Status:    int32(user.Status),
		CreatedAt: user.CreatedAt,
		RoleIds:   roleIDs,
	}
}

func wrapNotFoundError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	}
	return err
}

func wrapDuplicateError(err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique") {
		return errors.New("username or email already exists")
	}
	return err
}
