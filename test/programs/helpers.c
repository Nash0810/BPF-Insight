#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

SEC("kprobe/sys_execve")
int test_prog(struct pt_regs *ctx) {

    __u64 pid = bpf_get_current_pid_tgid();
    __u64 t = bpf_ktime_get_ns();

    if (pid != 0)
        return 1;
    else
        return 0;
}

char _license[] SEC("license") = "GPL";
