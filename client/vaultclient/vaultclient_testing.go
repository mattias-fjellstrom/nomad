// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vaultclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/nomad/helper/uuid"
)

// MockVaultClient is used for testing the vaultclient integration and is safe
// for concurrent access.
type MockVaultClient struct {

	// jwtTokens stores the tokens derived using the JWT flow.
	jwtTokens map[string]string

	// stoppedTokens tracks the tokens that have stopped renewing
	stoppedTokens []string

	// renewTokens are the tokens that have been renewed and their error
	// channels
	renewTokens map[string]chan error

	// renewTokenErrors is used to return an error when the RenewToken is called
	// with the given token
	renewTokenErrors map[string]error

	// deriveTokenErrors maps an allocation ID and tasks to an error when the
	// token is derived
	deriveTokenErrors map[string]map[string]error

	// deriveTokenWithJWTFn allows the caller to control the DeriveTokenWithJWT
	// function.
	deriveTokenWithJWTFn func(context.Context, JWTLoginRequest) (string, bool, int, error)

	// renewable determines if the tokens returned should be marked as renewable
	renewable bool

	duration int

	mu sync.Mutex
}

// NewMockVaultClient returns a MockVaultClient for testing
func NewMockVaultClient(_ string) (VaultClient, error) {
	return &MockVaultClient{renewable: true, duration: 30}, nil
}

func (vc *MockVaultClient) DeriveTokenWithJWT(ctx context.Context, req JWTLoginRequest) (string, bool, int, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.deriveTokenWithJWTFn != nil {
		return vc.deriveTokenWithJWTFn(ctx, req)
	}

	if vc.jwtTokens == nil {
		vc.jwtTokens = make(map[string]string)
	}

	token := uuid.Generate()
	if req.Role != "" {
		token = fmt.Sprintf("%s-%s", token, req.Role)
	}
	vc.jwtTokens[req.JWT] = token
	return token, vc.renewable, vc.duration, nil
}

func (vc *MockVaultClient) SetDeriveTokenError(allocID string, tasks []string, err error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.deriveTokenErrors == nil {
		vc.deriveTokenErrors = make(map[string]map[string]error, 10)
	}

	if _, ok := vc.deriveTokenErrors[allocID]; !ok {
		vc.deriveTokenErrors[allocID] = make(map[string]error, 10)
	}

	for _, task := range tasks {
		vc.deriveTokenErrors[allocID][task] = err
	}
}

func (vc *MockVaultClient) RenewToken(token string, interval int) (<-chan error, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if err, ok := vc.renewTokenErrors[token]; ok {
		return nil, err
	}

	renewCh := make(chan error)
	if vc.renewTokens == nil {
		vc.renewTokens = make(map[string]chan error, 10)
	}
	vc.renewTokens[token] = renewCh
	return renewCh, nil
}

func (vc *MockVaultClient) SetRenewTokenError(token string, err error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.renewTokenErrors == nil {
		vc.renewTokenErrors = make(map[string]error, 10)
	}

	vc.renewTokenErrors[token] = err
}

func (vc *MockVaultClient) StopRenewToken(token string) error {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.stoppedTokens = append(vc.stoppedTokens, token)
	return nil
}

func (vc *MockVaultClient) Start() {}

func (vc *MockVaultClient) Stop() {}

func (vc *MockVaultClient) SetRenewable(renewable bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.renewable = renewable
}

// JWTTokens returns the tokens generated suing the JWT flow.
func (vc *MockVaultClient) JWTTokens() map[string]string {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	return vc.jwtTokens
}

// StoppedTokens tracks the tokens that have stopped renewing
func (vc *MockVaultClient) StoppedTokens() []string {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	return vc.stoppedTokens
}

// RenewTokens are the tokens that have been renewed and their error
// channels
func (vc *MockVaultClient) RenewTokens() map[string]chan error {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	return vc.renewTokens
}

// RenewTokenErrCh returns the error channel for the given token renewal
// process.
func (vc *MockVaultClient) RenewTokenErrCh(token string) chan error {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	return vc.renewTokens[token]
}

// SetDeriveTokenWithJWTFn sets the function used to derive tokens using JWT.
func (vc *MockVaultClient) SetDeriveTokenWithJWTFn(f func(context.Context, JWTLoginRequest) (string, bool, int, error)) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.deriveTokenWithJWTFn = f
}
