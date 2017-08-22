import sys
import time
import getopt
import subprocess
from subprocess import Popen, PIPE

def main(argv):
        verbose = False;
        round = 20;
        try:
                opts, args = getopt.getopt(argv[1:], "vr:", ["verbose", "round="])
        except getopt.GetoptError:
                usage()
                sys.exit(2)
        for opt, arg in opts:
                if opt in ("-v", "--verbose"):
                        verbose = True;
                elif opt in ("-r", "--round"):
                        round = int(arg)
        doRounds(verbose, round)
        return 0

def usage():
        print("%s [-v] [-r round]" % (sys.argv[0]))
        return

def doRounds(verbose, round):
        containers = []
        stopJob = []
        removeJob = []
        startCommand = ["docker", "run", "-d", "reinblau/php-apache2",  "/usr/sbin/apache2ctl", "-D", "FOREGROUND"]
        startTime = time.time()

        # start containers
        i = 0
        while i < round:
                app = subprocess.Popen(startCommand, stdout=PIPE)
                app.wait()
                out = app.stdout.read().rstrip()
                if (verbose):
                        nTime = time.time()
                        print("start %d: t= %.1f c= %s" % (i, nTime - startTime, out))
                containers.append(out);
                i = i + 1
        midTime = time.time()
        if (verbose):
                print("done start. elapsed: %.1f" % (midTime - startTime))

        # stop containers
        i = 0
        for container in containers:
                appstop = subprocess.Popen(["docker", "stop", container], stdout=PIPE);
                stopJob.append(appstop)
                if (verbose):
                        nTime = time.time()
                        print("stop %d: t= %.1f c= %s" % (i, nTime - startTime, container))
                i = i + 1

        i = 0
        for job in stopJob:
	        job.wait()
                if (verbose):
                        nTime = time.time()
                        print("stoped %d: t= %.1f c= %s" % (i, nTime - startTime, container))
                i = i + 1
        mid2Time = time.time()
        if (verbose):
                print("done stop. elapsed: %.1f" % (mid2Time - midTime))


        # rm containers
        i = 0
        for container in containers:
                appstop = subprocess.Popen(["docker", "rm", "-f", container], stdout=PIPE);
                removeJob.append(appstop)
                if (verbose):
                        nTime = time.time()
                        print("remove %d: t= %.1f c= %s" % (i, nTime - startTime, container))
                i = i + 1

        i = 0
        for job in removeJob:
	        job.wait()
                if (verbose):
                        nTime = time.time()
                        print("removed %d: t= %.1f c= %s" % (i, nTime - startTime, container))
                i = i + 1

        endTime = time.time()

        print("done start/stop. %d rounds. elapsed-time: %.1f + %.1f + %.1f = %.1f" %
              (round, midTime - startTime, mid2Time -midTime, endTime - mid2Time, endTime - startTime))
        return


if __name__ == "__main__":
        main(sys.argv)
