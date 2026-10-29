; Independent x64 call-boundary observer; no Cathedral-generated instructions.
EXTERN ProbeMixedBody:PROC
EXTERN observed:QWORD
EXTERN forwardTarget:QWORD
EXTERN savedVector:OWORD
EXTERN seedDouble:QWORD
PUBLIC ProbeMixedEntry
PUBLIC ProbeRoundForward
PUBLIC ProbeSaved
.code
ProbeMixedEntry PROC
    mov r10, rsp
    and r10, 15
    mov observed[0], r10
    mov observed[8], rcx
    mov observed[16], rdx
    mov observed[24], r8
    movq r10, xmm3
    mov observed[32], r10
    mov r10, [rsp+40]
    mov observed[40], r10
    mov r10, [rsp+48]
    mov observed[48], r10
    mov r10, [rsp+56]
    mov observed[56], r10
    mov r10, 0102030405060708h
    mov [rsp+8], r10
    mov [rsp+16], r10
    mov [rsp+24], r10
    mov [rsp+32], r10
    jmp ProbeMixedBody
ProbeMixedEntry ENDP
ProbeRoundForward PROC
    mov observed[64], rdx
    mov observed[72], rcx
    jmp QWORD PTR forwardTarget
ProbeRoundForward ENDP
ProbeSaved PROC
    sub rsp, 264
    mov [rsp+32], rbx
    mov [rsp+40], rbp
    mov [rsp+48], rsi
    mov [rsp+56], rdi
    mov [rsp+64], r12
    mov [rsp+72], r13
    mov [rsp+80], r14
    mov [rsp+88], r15
    movdqu [rsp+96], xmm6
    movdqu [rsp+112], xmm7
    movdqu [rsp+128], xmm8
    movdqu [rsp+144], xmm9
    movdqu [rsp+160], xmm10
    movdqu [rsp+176], xmm11
    movdqu [rsp+192], xmm12
    movdqu [rsp+208], xmm13
    movdqu [rsp+224], xmm14
    movdqu [rsp+240], xmm15
    mov rax, rcx
    mov rbx, 1122334455667788h
    mov rbp, 2233445566778899h
    mov rsi, 33445566778899AAh
    mov rdi, 445566778899AABBh
    mov r12, 5566778899AABBCCh
    mov r13, 66778899AABBCCDDh
    mov r14, 778899AABBCCDDEEh
    mov r15, 123456789ABCDEF0h
    movdqu xmm6, savedVector
    movdqu xmm7, savedVector
    movdqu xmm8, savedVector
    movdqu xmm9, savedVector
    movdqu xmm10, savedVector
    movdqu xmm11, savedVector
    movdqu xmm12, savedVector
    movdqu xmm13, savedVector
    movdqu xmm14, savedVector
    movdqu xmm15, savedVector
    movsd xmm0, seedDouble
    call rax
    movq r10, xmm0
    mov r11, 402E000000000000h
    cmp r10, r11
    jne failed
    mov r11, 1122334455667788h
    cmp rbx, r11
    jne failed
    mov r11, 2233445566778899h
    cmp rbp, r11
    jne failed
    mov r11, 33445566778899AAh
    cmp rsi, r11
    jne failed
    mov r11, 445566778899AABBh
    cmp rdi, r11
    jne failed
    mov r11, 5566778899AABBCCh
    cmp r12, r11
    jne failed
    mov r11, 66778899AABBCCDDh
    cmp r13, r11
    jne failed
    mov r11, 778899AABBCCDDEEh
    cmp r14, r11
    jne failed
    mov r11, 123456789ABCDEF0h
    cmp r15, r11
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm6
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm7
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm8
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm9
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm10
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm11
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm12
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm13
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm14
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    movdqu xmm1, savedVector
    pcmpeqb xmm1, xmm15
    pmovmskb r10d, xmm1
    cmp r10d, 65535
    jne failed
    xor eax, eax
    jmp restore
failed:
    mov eax, 1
restore:
    movdqu xmm6, [rsp+96]
    movdqu xmm7, [rsp+112]
    movdqu xmm8, [rsp+128]
    movdqu xmm9, [rsp+144]
    movdqu xmm10, [rsp+160]
    movdqu xmm11, [rsp+176]
    movdqu xmm12, [rsp+192]
    movdqu xmm13, [rsp+208]
    movdqu xmm14, [rsp+224]
    movdqu xmm15, [rsp+240]
    mov rbx, [rsp+32]
    mov rbp, [rsp+40]
    mov rsi, [rsp+48]
    mov rdi, [rsp+56]
    mov r12, [rsp+64]
    mov r13, [rsp+72]
    mov r14, [rsp+80]
    mov r15, [rsp+88]
    add rsp, 264
    ret
ProbeSaved ENDP
END
