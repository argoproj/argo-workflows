#include <signal.h>
#include <stdio.h>
#include <sys/types.h>

int main(int argc, char **argv) {
  printf("killing %s with %s\n", argv[2], argv[1]);
  if (argv[1] == "15") {
    return kill(atoi(argv[2]), SIGTERM);
  } else {
    return kill(atoi(argv[2]), SIGKILL);
  }
}