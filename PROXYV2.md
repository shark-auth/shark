JWT Baking como Estado: El Control Plane (backend de Shark) emite JWTs que contienen el estado completo del usuario (roles, tier de suscripción free/pro). El proxy (Data Plane) evalúa las rutas basándose exclusivamente en la criptografía del JWT, sin hacer llamadas a la DB.

Edge Paywalls Estáticos: Si el proxy lee que la ruta requiere tier: pro y el JWT dice tier: free, intercepta la petición y sirve el componente de "Upgrade" pre-renderizado.

CLI-Native & SKILL.md (Agent-Built Auth): Formalizar el protocolo de comandos. El CLI no solo interactúa con el dashboard, sino que expone una interfaz predecible para que agentes autónomos (vía Cursor, Claude, etc.) puedan aprovisionar toda la infraestructura de autenticación de un SaaS ejecutando scripts automatizados.

Fase 2: "Magic Onboarding" y White-labeling (Corto/Medio Plazo)
Aquí es donde la experiencia de producto brilla y donde puedes delegar la abstracción visual para mantener el diseño pulido y profesional.

Inferencia de Estilos vía CLI: En lugar de un scraper en el backend, cuando el desarrollador ejecuta shark init, el CLI analiza el frontend local del cliente, extrae paletas de colores (preferencia por dark-mode, tipografías limpias) y sube esa configuración al Control Plane.

Renderizado Dinámico en el Edge: El proxy inyecta esas variables de diseño en los componentes alojados (Login/Signup/Paywalls) al vuelo. El usuario final ve una transición transparente y cohesiva sin que el desarrollador haya escrito una línea de CSS en SharkAuth.

Perfeccionamiento del Handoff: Estandarizar el inyector del X-Shark-Profile (en Base64) para que el backend del cliente reciba un bloque de identidad inmutable y verificable.

Fase 3: El "Agentic Router" (El Moonshot)

Bifurcación Heurística de Tráfico: El proxy evalúa los headers y patrones de petición en menos de 5ms para determinar si el cliente es un navegador (humano) o un script/LLM (agente).

Gatekeeping Financiero por Tokens: Para rutas de API consumidas por agentes, el proxy deja de contar peticiones por segundo (rate-limiting tradicional) y comienza a hacer un shallow parse de los payloads JSON para contabilizar tokens, interceptando la llamada si el agente se quedó sin presupuesto.

Auth Nativa para Máquinas: Implementar flujos de identidad máquina-a-máquina (M2M) directamente en el proxy, permitiendo que agentes de IA se autentiquen y obtengan permisos temporales para interactuar con el SaaS del cliente.

Fase 4: Ejecución Arbitraria (Largo Plazo)
Cuando el sistema alcance una escala donde las reglas CEL ya no sean suficientes para las demandas de los clientes "Enterprise".

WASM Plugins en el Edge: Inspirado en tecnologías de observabilidad y proxies de alto rendimiento, permitir que los desarrolladores compilen su lógica de negocio hiper-específica en binarios WebAssembly. El proxy de SharkAuth ejecuta estos plugins en un sandbox seguro en el Edge para tomar decisiones de ruteo y acceso personalizadas.

Sincronización In-Memory en Tiempo Real: Implementar un sistema Pub/Sub (ej. gRPC streams o NATS) para que el Control Plane empuje invalidaciones de caché al proxy de forma instantánea, logrando un RBAC dinámico sin sacrificar latencia.
