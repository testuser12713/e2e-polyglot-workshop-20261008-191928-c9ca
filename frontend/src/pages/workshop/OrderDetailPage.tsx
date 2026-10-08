import { useParams } from "react-router-dom";

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();

  return (
    <section>
      <h1 className="page__title">Auftrag {id}</h1>
      <p className="page__lead">Positionen, Statusverlauf und Statuswechsel für diesen Auftrag.</p>
    </section>
  );
}
