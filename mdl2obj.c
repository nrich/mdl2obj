/*
 * mdl2obj
 *
 * Copyright (C) 2016 Florian Zwoch <fzwoch@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

#include <glib.h>
#include <string.h>

struct vec3_s {
	gfloat x;
	gfloat y;
	gfloat z;
};

struct mdl_s {
	guint32 id;
	guint32 version;
	struct vec3_s scale;
	struct vec3_s origin;
	gfloat radius;
	struct vec3_s offsets;
	guint32 num_skins;
	guint32 skin_width;
	guint32 skin_height;
	guint32 num_verts;
	guint32 num_triangles;
	guint32 num_frames;
	guint32 sync_type;
	guint32 flags;
	gfloat size;
};

struct skin_s {
	guint32 type;
};

struct skin_group_s {
	guint32 type;
	guint32 num_skins;
	gfloat time;
};

struct stvert_s {
	guint32 on_seam;
	guint32 s;
	guint32 t;
};

struct triangle_s {
	guint32 front;
	guint32 vertex[3];
};

struct vert_s {
	guint8 v[3];
	guint8 normal;
};

struct frame_s {
	guint32 type;
	struct vert_s min;
	struct vert_s max;
	gchar name[16];
};

int main(int argc, char **argv)
{
	GError *err = NULL;
	gchar *buf = NULL;
	gsize buf_len = 0;
	
	if (argc != 2)
	{
		g_print("usage: %s <model.mdl>\n", argv[0]);
		return 1;
	}
	
	g_file_get_contents(argv[1], &buf, &buf_len, &err);
	if (err != NULL)
	{
		g_error("%s", err->message);
		g_error_free(err);
		
		return 1;
	}
	
	struct mdl_s *mdl = buf;
	struct skin_s *skin = mdl + 1;
	
	for (gint i = 0; i < mdl->num_skins; i++)
	{
		if (skin->type != 0)
		{
			struct skin_group_s *skin_group = skin;
			
			skin = (void*)skin + sizeof(struct skin_group_s) + skin_group->num_skins * mdl->skin_width * mdl->skin_height;
		}
		else
		{
			skin = (void*)skin + sizeof(struct skin_s) + mdl->skin_width * mdl->skin_height;
		}
	}
	
	struct stvert_s *stvert = skin;
	struct triangle_s *triangle = stvert + mdl->num_verts;
	struct frame_s *frame = triangle + mdl->num_triangles;
	
	GString *obj = g_string_new(NULL);
	
	gchar *mdl_name = g_path_get_basename(argv[1]);
	mdl_name[strcspn(mdl_name, ".")] = '\0';
	
	g_string_append_printf(obj, "newmtl %s\n", mdl_name);
	g_string_append_printf(obj, "Ka 1 1 1\n");
	g_string_append_printf(obj, "Kd 1 1 1\n");
	g_string_append_printf(obj, "Ks 0 0 0\n");
	g_string_append_printf(obj, "Tr 1\n");
	g_string_append_printf(obj, "illum 1\n");
	g_string_append_printf(obj, "Ns 0\n");
	g_string_append_printf(obj, "map_Kd textures/%s.jpg\n", mdl_name);
	
	gchar *out_file = g_strdup_printf("%s.mtl", mdl_name);
	
	g_file_set_contents(out_file, obj->str, obj->len, &err);
	if (err != NULL)
	{
		g_error("%s", err->message);
		g_error_free(err);
		
		return 1;
	}
	
	for (gint k = 0; k < mdl->num_frames; k++)
	{
		struct vert_s *vert = frame + 1;
		
		if (frame->type != 0)
		{
			frame = (void*)frame + 2 * sizeof(struct vert_s) + sizeof(gfloat);
		}
		
		frame = (void*)frame + sizeof(struct frame_s) + mdl->num_verts * sizeof(struct vert_s);
		
		g_string_printf(obj, "mtllib %s.mtl\n", mdl_name);
		g_string_append_printf(obj, "usemtl %s\n", mdl_name);
		
		for (gint i = 0; i < mdl->num_verts; i++)
		{
			g_string_append_printf(obj, "v %g %g %g\n",
								   mdl->scale.x * vert[i].v[0] + mdl->origin.x,
								   mdl->scale.y * vert[i].v[1] + mdl->origin.y,
								   mdl->scale.z * vert[i].v[2] + mdl->origin.z);
		}

		gboolean needs_hack = FALSE;

		for (gint i = 0; i < mdl->num_verts; i++)
		{
			g_string_append_printf(obj, "vt %g %g\n",
								   (float) stvert[i].s / mdl->skin_width,
								   1 - (float) stvert[i].t / mdl->skin_height);

			if (stvert[i].on_seam != 0)
			{
				needs_hack = TRUE;
			}
		}

		if (needs_hack == TRUE)
		{
			for (gint i = 0; i < mdl->num_verts; i++)
			{
				if (stvert[i].on_seam == 0)
				{
					g_string_append_printf(obj, "vt 0 0\n"); // can never be used
				}
				else
				{
					g_string_append_printf(obj, "vt %g %g\n",
										(float) (stvert[i].s + mdl->skin_width / 2) / mdl->skin_width,
										1 - (float) stvert[i].t / mdl->skin_height);
				}
			}
		}
		
		for (gint i = 0; i < mdl->num_triangles; i++)
		{
			g_string_append_printf(obj, "f %d/%d %d/%d %d/%d\n",
								   triangle[i].vertex[0] + 1,
								   triangle[i].vertex[0] + 1 + (triangle[i].front == 0 && stvert[triangle[i].vertex[0]].on_seam != 0 ? mdl->num_verts : 0),
								   triangle[i].vertex[2] + 1,
								   triangle[i].vertex[2] + 1 + (triangle[i].front == 0 && stvert[triangle[i].vertex[2]].on_seam != 0 ? mdl->num_verts : 0),
								   triangle[i].vertex[1] + 1,
								   triangle[i].vertex[1] + 1 + (triangle[i].front == 0 && stvert[triangle[i].vertex[1]].on_seam != 0 ? mdl->num_verts : 0));
		}
		
		if (mdl->num_frames == 1)
		{
			out_file = g_strdup_printf("%s.obj", mdl_name);
		}
		else
		{
			out_file = g_strdup_printf("%s_%d.obj", mdl_name, k);
		}
		
		g_file_set_contents(out_file, obj->str, obj->len, &err);
		if (err != NULL)
		{
			g_error("%s", err->message);
			g_error_free(err);
			
			return 1;
		}
		
		g_free(out_file);
	}
	
	g_string_free(obj, TRUE);
	g_free(buf);
	
	return 0;
}
