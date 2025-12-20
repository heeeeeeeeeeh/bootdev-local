.PHONY: */* */*/*

*/*/*:
	cd $@ && python main_test.py >/dev/null
	cd $@ && python main_test.py && python main_test.py  | grep -q "============= FAIL ==============" && exit 1 || exit 0

*/*:
	cd $@ && python main_test.py  |  grep -q "============= FAIL ==============" && exit 1 || exit 0
