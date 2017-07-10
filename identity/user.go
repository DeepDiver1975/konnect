/*
 * Copyright 2017 Kopano and its licensors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package identity

import (
	"github.com/dgrijalva/jwt-go"
)

// User defines a most simple user with an id.
type User interface {
	Subject() string
}

// UserWithEmail is a User with Email.
type UserWithEmail interface {
	User
	Email() string
	EmailVerified() bool
}

// UserWithProfile is a User with Name.
type UserWithProfile interface {
	User
	Name() string
}

// UserWithID is a User with a numeric id.
type UserWithID interface {
	User
	ID() int64
}

// UserWithClaims is A User with jwt claims.
type UserWithClaims interface {
	User
	Claims() jwt.MapClaims
}
