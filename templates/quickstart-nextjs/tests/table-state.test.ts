import assert from "node:assert/strict";
import test from "node:test";
import { tablePageState, nextTableSort } from "../lib/table-state";

test("table pagination handles empty data, page boundaries and invalid inputs", () => {
 assert.deepEqual(tablePageState(1,10,0), {page:1,pages:1,previous:false,next:false});
 assert.deepEqual(tablePageState(2,10,21), {page:2,pages:3,previous:true,next:true});
 assert.deepEqual(tablePageState(9,10,21), {page:3,pages:3,previous:true,next:false});
 assert.deepEqual(tablePageState(-1,0,-2), {page:1,pages:1,previous:false,next:false});
});
test("table sorting starts ascending and toggles only the selected column", () => {
 assert.deepEqual(nextTableSort(undefined,"date"),{key:"date",direction:"asc"});
 assert.deepEqual(nextTableSort({key:"date",direction:"asc"},"date"),{key:"date",direction:"desc"});
 assert.deepEqual(nextTableSort({key:"date",direction:"desc"},"amount"),{key:"amount",direction:"asc"});
});
