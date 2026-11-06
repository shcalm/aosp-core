import unittest
import pkgutil
import sys

if __name__ == '__main__':
    test_modules = [
        mod.name
        for mod in pkgutil.walk_packages()
        if mod.name.startswith('tests.test_')
    ]

    suite = unittest.defaultTestLoader.loadTestsFromNames(test_modules)

    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    # Exit with non-zero code if any test failed
    sys.exit(not result.wasSuccessful())
