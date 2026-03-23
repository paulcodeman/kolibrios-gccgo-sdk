    .text
    .globl runtime_swapcontext
    .type runtime_swapcontext, @function
runtime_swapcontext:
    movl 4(%esp), %eax
    movl 8(%esp), %edx

    movl %ebx, 0(%eax)
    movl %esi, 4(%eax)
    movl %edi, 8(%eax)
    movl %ebp, 12(%eax)
    movl (%esp), %ecx
    movl %ecx, 20(%eax)
    leal 4(%esp), %ecx
    movl %ecx, 16(%eax)

    movl 0(%edx), %ebx
    movl 4(%edx), %esi
    movl 8(%edx), %edi
    movl 12(%edx), %ebp
    movl 16(%edx), %esp
    movl 20(%edx), %ecx
    jmp *%ecx

    .size runtime_swapcontext, .-runtime_swapcontext

    .globl runtime_loadcontext
    .type runtime_loadcontext, @function
runtime_loadcontext:
    movl 4(%esp), %edx

    movl 0(%edx), %ebx
    movl 4(%edx), %esi
    movl 8(%edx), %edi
    movl 12(%edx), %ebp
    movl 16(%edx), %esp
    movl 20(%edx), %ecx
    jmp *%ecx

    .size runtime_loadcontext, .-runtime_loadcontext

    .globl runtime_kolibri_thread_entry
    .type runtime_kolibri_thread_entry, @function
runtime_kolibri_thread_entry:
    movl (%esp), %eax
    andl $0xFFFFFFF0, %esp
    subl $8, %esp
    push %eax
    call runtime_m_start
    addl $12, %esp
    call go_0kos.ExitRaw
    ret

    .size runtime_kolibri_thread_entry, .-runtime_kolibri_thread_entry

    .globl runtime_libgo_thread_entry
    .type runtime_libgo_thread_entry, @function
runtime_libgo_thread_entry:
    movl (%esp), %eax
    andl $0xFFFFFFF0, %esp
    subl $8, %esp
    push %eax
    call runtime_mstart
    addl $12, %esp
    call go_0kos.ExitRaw
    ret

    .size runtime_libgo_thread_entry, .-runtime_libgo_thread_entry
