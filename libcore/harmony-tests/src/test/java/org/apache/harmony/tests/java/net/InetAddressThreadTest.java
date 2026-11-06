/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.apache.harmony.tests.java.net;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.InetAddress;
import java.net.UnknownHostException;
import java.util.ArrayList;
import java.util.EnumMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentLinkedQueue;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

import tests.support.Support_Configuration;

import org.junit.Assume;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class InetAddressThreadTest {

    /*
     * REP_NUM can be adjusted if desired. Since this error is
     * non-deterministic it may not always occur. Setting REP_NUM higher,
     * increases the chances of an error being detected, but causes the test
     * to take longer. Because the Java threads spend a lot of time
     * performing operations other than running the native code that may not
     * be threadsafe, it is quite likely that several thousand iterations
     * will elapse before the first error is detected.
     */
    private static final int REP_NUM = 20000;
    private static final long THREAD_TIMEOUT_MS = 240000;
    private static final String LOCALHOST_NAME = "localhost";
    private static final String LOCALHOST_IP = "127.0.0.1";
    private static final int NUM_LOOKUP_TYPES = 2;

    private enum LookupType {
        BY_NAME {
            @Override
            public String toString() {
                return "gethostbyname";
            }
        },
        BY_ADDRESS {
            @Override
            public String toString() {
                return "gethostbyaddr";
            }
        };
    };

    private final CountDownLatch startedLatch = new CountDownLatch(1);

    private final Map<LookupType, AtomicBoolean> someoneDone = new EnumMap<>(LookupType.class);
    {
        for (LookupType type : LookupType.values()) {
            someoneDone.put(type, new AtomicBoolean(false));
        }
    }

    private final ConcurrentLinkedQueue<String> errorMessages = new ConcurrentLinkedQueue<>();

    private String originalTtl;

    /**
     * This class is used to test inet_ntoa, gethostbyaddr and gethostbyname
     * functions in the VM to make sure they're threadsafe. getByName will cause
     * the gethostbyname function to be called. getHostName will cause the
     * gethostbyaddr to be called. getHostAddress will cause inet_ntoa to be
     * called.
     */
    class ThreadsafeTestThread extends Thread {
        private final String lookupName;
        private final InetAddress expectedResult;
        private final LookupType testType;

        public ThreadsafeTestThread(String name, String lookupName,
                InetAddress expectedResult, LookupType testType) {
            super(name);
            this.lookupName = lookupName;
            this.expectedResult = expectedResult;
            this.testType = testType;
        }

        public void run() {
            try {
                String expectedName = expectedResult.getHostName();
                String expectedAddress = expectedResult.getHostAddress();
                long startTime = System.currentTimeMillis();

                if (!startedLatch.await(10, TimeUnit.SECONDS)) {
                    reportError("Start latch timeout");
                    return;
                }

                for (int i = 0; i < REP_NUM; i++) {
                    if (hasStopCondition(i, startTime)) {
                        break;
                    }

                    try {
                        InetAddress ia = InetAddress.getByName(lookupName);
                        String hostName = ia.getHostName();
                        String hostAddress = ia.getHostAddress();

                        // Intentionally not looking for exact name match so that
                        // the test works across different platforms that may or
                        // may not include a domain suffix on the hostname
                        if (!hostName.startsWith(expectedName)) {
                            reportUnexpectedValue("getHostName", i,
                                    expectedName, hostName);
                            break;
                        }
                        // IP addresses should match exactly
                        if (!expectedAddress.equals(hostAddress)) {
                            reportUnexpectedValue("getHostAddress", i,
                                    expectedAddress, hostAddress);
                            break;
                        }
                    } catch (UnknownHostException e) {
                        reportError(String.format("UnknownHostException for %s (iteration=%d):\n%s",
                                    lookupName, i, exceptionToString(e)));
                    }

                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                reportError(String.format("Interrupted:\n%s", exceptionToString(e)));
            } catch (Throwable e) {
                reportError(String.format("Unexpected error:\n%s", exceptionToString(e)));
            } finally {
                // Signal that this thread type has completed.
                someoneDone.get(testType).set(true);
            }
        }

        private boolean hasStopCondition(int iteration, long startTime) {
            if (someoneDone.get(testType).get() || !errorMessages.isEmpty()) {
                return true;
            }
            if ((iteration % 25) == 0 &&
                    System.currentTimeMillis() - startTime > THREAD_TIMEOUT_MS) {
                System.out.println("Exiting due to time limitation after "
                        + iteration + " iterations");
                return true;
            }
            return false;
        }

        private void reportError(String message) {
            errorMessages.add(String.format("[%s - %s]: %s", getName(),
                        testType.toString(), message));
        }

        private void reportUnexpectedValue(String fn, int iteration,
                String expected, String actual) {
            reportError(String.format("(iteration=%d): %s() for %s returned %s instead of %s",
                iteration, fn, lookupName, actual, expected));
        }

        private String exceptionToString(Throwable e) {
            StringWriter sw = new StringWriter();
            e.printStackTrace(new PrintWriter(sw));
            return sw.toString();
        }
    }

    @Before
    public void setUp() {
        // Make sure there is no caching
        originalTtl = System.getProperty("networkaddress.cache.ttl");
        System.setProperty("networkaddress.cache.ttl", "0");
        InetAddress.clearDnsCache();
    }

    @After
    public void tearDown() {
        // restore the old value of the property
        if (originalTtl == null) {
            // setting the property to -1 has the same effect as having the
            // property be null
            System.setProperty("networkaddress.cache.ttl", "-1");
        } else {
            System.setProperty("networkaddress.cache.ttl", originalTtl);
        }
    }

    /**
     * java.net.InetAddress#getHostName()
     */
    @Test
    public void test_getHostName() throws Exception {
        // Test for method java.lang.String java.net.InetAddress.getHostName()
        InetAddress[] lookups = new InetAddress[NUM_LOOKUP_TYPES];
        String[] lookupNames = {LOCALHOST_NAME, LOCALHOST_IP};
        try {
            for (int i = 0; i < NUM_LOOKUP_TYPES; i++) {
                lookups[i] = InetAddress.getByName(lookupNames[i]);
            }
        } catch (UnknownHostException ex) {
            Assume.assumeNoException("Skipping test: " + LOCALHOST_NAME + " or " +
                    LOCALHOST_IP + " cannot be resolved", ex);
            return;
        }

        for(int i = 0; i < NUM_LOOKUP_TYPES; i++) {
            assertEquals(LOCALHOST_IP, lookups[i].getHostAddress());
            String initialHostName = lookups[i].getHostName();
            Assume.assumeTrue(String.format("Skipping test: Initial reverse lookup for "+
                        "%s (from %s) returned '%s', expected '%s'. Environment issue?",
                        lookups[i].getHostAddress(), lookupNames[i],
                        initialHostName, LOCALHOST_NAME),
                    LOCALHOST_NAME.equals(initialHostName));
        }


        // Test for threadsafety
        List<Thread> threads = new ArrayList<>();
        for (int i = 0; i < NUM_LOOKUP_TYPES; i++) {
            threads.add(new ThreadsafeTestThread(
                "ByName-" + i, LOCALHOST_NAME, lookups[i], LookupType.BY_NAME));
            threads.add(new ThreadsafeTestThread(
                "ByAddr-" + i, LOCALHOST_IP, lookups[i], LookupType.BY_ADDRESS));
        }

        for (Thread t : threads) {
            t.start();
        }
        startedLatch.countDown();
        for (Thread t : threads) {
            t.join();
        }

        assertTrue("Test failed with errors:\n" + String.join("\n", errorMessages),
                errorMessages.isEmpty());
    }
}
