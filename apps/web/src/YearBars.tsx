import { YearCount } from "./yearBars";

const MIN_BAR_PERCENT = 2;

export function YearBars({ data, caption }: { data: YearCount[]; caption: string }) {
  const max = Math.max(...data.map((d) => d.count));
  const peak = data.findIndex((d) => d.count === max);
  const tickEvery = data.length > 10 ? 2 : 1;
  return (
    <figure className="year-bars">
      <div className="year-bars-plot">
        {data.map((d, i) => (
          <div key={d.year} className="year-bar" title={`${d.year}: ${d.count.toLocaleString("pt-BR")} atos`}>
            <span className="year-bar-fill" style={{ height: `${Math.max(MIN_BAR_PERCENT, (d.count / max) * 100)}%` }}>
              {i === peak && <span className="year-bar-value">{d.count.toLocaleString("pt-BR")}</span>}
            </span>
          </div>
        ))}
      </div>
      <div className="year-bars-axis" aria-hidden="true">
        {data.map((d, i) => <span key={d.year}>{i % tickEvery === 0 ? d.year : ""}</span>)}
      </div>
      <table className="visually-hidden">
        <caption>{caption}</caption>
        <tbody>
          {data.map((d) => <tr key={d.year}><th scope="row">{d.year}</th><td>{d.count}</td></tr>)}
        </tbody>
      </table>
    </figure>
  );
}
