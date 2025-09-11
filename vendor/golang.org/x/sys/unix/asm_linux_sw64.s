// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && sw64 && gc

#include "textflag.h"

//
// System calls for sw64, Linux
//
// Just jump to package syscall's implementation for all these functions.
// The runtime may know about them.

#define SYSCALL SYS_CALL_B $131

TEXT ·Syscall(SB),NOSPLIT|NOFRAME,$0-56
	JMP	syscall·Syscall(SB)

TEXT ·Syscall6(SB),NOSPLIT|NOFRAME,$0-80
	JMP	syscall·Syscall6(SB)

TEXT ·SyscallNoError(SB),NOSPLIT,$0-48
	CALL	runtime·entersyscall(SB)
	LDL	R16, a1+8(FP)
	LDL	R17, a2+16(FP)
	LDL	R18, a3+24(FP)
	LDI	R19, ZERO
	LDI	R20, ZERO
	LDI	R21, ZERO
	LDL	R0, trap+0(FP)	// syscall entry
	SYSCALL
	STL	R0, r1+32(FP)
	STL	R20, r2+40(FP)
	CALL	runtime·exitsyscall(SB)
	RET

TEXT ·RawSyscall(SB),NOSPLIT|NOFRAME,$0-56
	JMP	syscall·RawSyscall(SB)

TEXT ·RawSyscall6(SB),NOSPLIT|NOFRAME,$0-80
	JMP	syscall·RawSyscall6(SB)

TEXT ·RawSyscallNoError(SB),NOSPLIT|NOFRAME,$0-48
	LDL	R16, a1+8(FP)
	LDL	R17, a2+16(FP)
	LDL	R18, a3+24(FP)
	LDI	R19, ZERO
	LDI	R20, ZERO
	LDI	R21, ZERO
	LDL	R0, trap+0(FP)	// syscall entry
	SYSCALL
	STL	R0, r1+32(FP)
	STL	R20, r2+40(FP)
	RET
