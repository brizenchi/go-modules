import assert from "node:assert/strict";
import test from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { DataTable } from "../components/data-table";

test("shared table renders accessible captions, columns and loading without a false empty state", () => {
 const html=renderToStaticMarkup(<DataTable caption="Credit history" columns={[{key:"id",heading:"ID",cell:(row:{id:number})=>row.id}]} rows={[]} rowKey={row=>row.id} loading loadingLabel="Loading history" empty="No records" />);
 assert.match(html,/caption/); assert.match(html,/scope="col"/); assert.match(html,/aria-busy="true"/); assert.match(html,/Loading history/); assert.doesNotMatch(html,/No records/);
});
test("shared table escapes content, marks sorting and disables empty pagination", () => {
 const html=renderToStaticMarkup(<DataTable caption="History" columns={[{key:"text",heading:"Text",sortable:true,cell:(row:{id:number;text:string})=>row.text}]} rows={[{id:1,text:"<script>private</script>"}]} rowKey={row=>row.id} loadingLabel="Loading" empty="Empty" sort={{key:"text",direction:"asc"}} onSort={()=>{}} pagination={{page:1,limit:10,total:1,onPage:()=>{},previousLabel:"Previous",nextLabel:"Next",label:(p,n)=>`${p}/${n}`}} />);
 assert.match(html,/&lt;script&gt;/); assert.doesNotMatch(html,/<script>/); assert.match(html,/aria-sort="ascending"/); assert.equal((html.match(/disabled=""/g)||[]).length,2);
});
