#include <assert.h>
#include <stdio.h>

#include "lfi_libadd_bin_box.h"

int add(int, int);
int sub(int, int);

int main()
{
    int r = LFI_CALL(add, 10, 32);
    printf("add(10, 32) = %d\n", r);
    assert(r == 42);
    int s = LFI_CALL(sub, 10, 32);
    printf("sub(10, 32) = %d\n", s);
    assert(s == -22);
}
